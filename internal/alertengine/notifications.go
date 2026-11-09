package alertengine

import (
	"context"
	"database/sql"
	"fmt"
)

// DeliverNotifications copies new centralized alerts into each active user's
// inbox. The bridge makes retries and concurrent workers idempotent.
func DeliverNotifications(ctx context.Context, db *sql.DB, limit int) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("notification database is required")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.QueryContext(ctx, `SELECT a.id,u.id,COALESCE(NULLIF(a.title,''),'Alert Kendaraan'),CONCAT(COALESCE(NULLIF(v.plate_number,''),NULLIF(v.vehicle_name,''),CONCAT('Vehicle #',a.vehicle_id)),': ',COALESCE(NULLIF(a.description,''),'Alert kendaraan memerlukan perhatian.')) FROM alert_events a JOIN vehicles v ON v.id=a.vehicle_id JOIN users u ON u.is_active=1 AND u.deleted_at IS NULL LEFT JOIN alert_inbox_deliveries d ON d.alert_id=a.id AND d.user_id=u.id WHERE d.id IS NULL ORDER BY a.id,u.id LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	type delivery struct {
		alertID, userID int64
		title, body     string
	}
	var pending []delivery
	for rows.Next() {
		var item delivery
		if err = rows.Scan(&item.alertID, &item.userID, &item.title, &item.body); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, item := range pending {
		tx, e := db.BeginTx(ctx, nil)
		if e != nil {
			return delivered, e
		}
		var exists int
		e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM alert_inbox_deliveries WHERE alert_id=? AND user_id=? FOR UPDATE`, item.alertID, item.userID).Scan(&exists)
		if e == nil && exists == 0 {
			result, insertErr := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,body,target) VALUES(?,?,?,'/#alert-center')`, item.userID, item.title, item.body)
			e = insertErr
			if e == nil {
				notificationID, idErr := result.LastInsertId()
				e = idErr
				if e == nil {
					_, e = tx.ExecContext(ctx, `INSERT INTO alert_inbox_deliveries(alert_id,user_id,notification_id) VALUES(?,?,?)`, item.alertID, item.userID, notificationID)
				}
			}
		}
		if e != nil {
			tx.Rollback()
			return delivered, e
		}
		if e = tx.Commit(); e != nil {
			return delivered, e
		}
		if exists == 0 {
			delivered++
		}
	}
	return delivered, nil
}
