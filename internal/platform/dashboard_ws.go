package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"time"

	"golang.org/x/net/websocket"
)

type dashboardSocketEnvelope struct {
	SchemaVersion       int                     `json:"schemaVersion"`
	Type                string                  `json:"type"`
	Cursor              uint64                  `json:"cursor"`
	AsOf                time.Time               `json:"asOf"`
	Vehicles            []dashboardVehicle      `json:"vehicles,omitempty"`
	Summary             *dashboardSummary       `json:"summary,omitempty"`
	Upserts             []dashboardVehicle      `json:"upserts,omitempty"`
	Removed             []string                `json:"removed,omitempty"`
	FuelSamples         []dashboardFuelSample   `json:"fuelSamples,omitempty"`
	Notifications       []dashboardNotification `json:"notifications,omitempty"`
	NotificationUpserts []dashboardNotification `json:"notificationUpserts,omitempty"`
	NotificationRemoved []string                `json:"notificationRemoved,omitempty"`
	Reason              string                  `json:"reason,omitempty"`
}

type dashboardNotification struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	Target    string  `json:"target"`
	ReadAt    *string `json:"readAt"`
	CreatedAt string  `json:"createdAt"`
}

type dashboardFuelSample struct {
	VehicleID     string   `json:"vehicleId"`
	SampleID      string   `json:"sampleId"`
	At            string   `json:"at"`
	ReceivedAt    *string  `json:"receivedAt"`
	Liters        *float64 `json:"liters"`
	Percent       *float64 `json:"percent"`
	Validity      string   `json:"validity"`
	CalibrationID *string  `json:"calibrationId"`
}

const dashboardSocketWriteTimeout = 5 * time.Second

func dashboardSocketSend(connection *websocket.Conn, envelope dashboardSocketEnvelope) error {
	if err := connection.SetWriteDeadline(time.Now().Add(dashboardSocketWriteTimeout)); err != nil {
		return err
	}
	return websocket.JSON.Send(connection, envelope)
}

func (s *Server) recordDashboardReplay(ctx context.Context, envelope dashboardSocketEnvelope) {
	if s.DB == nil || envelope.Type != "delta" {
		return
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	_, _ = s.DB.ExecContext(ctx, "INSERT INTO dashboard_replay_events(replay_cursor,event_kind,payload) VALUES(?, ?, ?)", envelope.Cursor, envelope.Type, payload)
}

func dashboardNextFreshness(vehicles []dashboardVehicle, now time.Time) time.Time {
	var next time.Time
	consider := func(raw *string) {
		if raw == nil {
			return
		}
		at, err := time.Parse(time.RFC3339Nano, *raw)
		if err != nil {
			return
		}
		deadline := at.UTC().Add(5 * time.Minute)
		if deadline.After(now) && (next.IsZero() || deadline.Before(next)) {
			next = deadline
		}
	}
	for _, vehicle := range vehicles {
		consider(vehicle.GatewayReceivedAt)
		consider(vehicle.GPS.MeasuredAt)
		consider(vehicle.Fuel.MeasuredAt)
		consider(vehicle.Fuel.ReceivedAt)
		consider(vehicle.SensorReceivedAt)
	}
	return next
}

func dashboardDelta(before, after []dashboardVehicle) ([]dashboardVehicle, []string) {
	prior := make(map[string]dashboardVehicle, len(before))
	next := make(map[string]dashboardVehicle, len(after))
	for _, row := range before {
		prior[row.ID] = row
	}
	upserts := make([]dashboardVehicle, 0)
	for _, row := range after {
		next[row.ID] = row
		if old, present := prior[row.ID]; !present || !reflect.DeepEqual(old, row) {
			upserts = append(upserts, row)
		}
	}
	removed := make([]string, 0)
	for id := range prior {
		if _, present := next[id]; !present {
			removed = append(removed, id)
		}
	}
	sort.Strings(removed)
	return upserts, removed
}

func dashboardFuelSamples(before, after []dashboardVehicle) []dashboardFuelSample {
	prior := make(map[string]*string, len(before))
	for _, vehicle := range before {
		prior[vehicle.ID] = vehicle.Fuel.SampleID
	}
	samples := make([]dashboardFuelSample, 0)
	for _, vehicle := range after {
		if vehicle.Fuel.SampleID == nil || vehicle.Fuel.MeasuredAt == nil {
			continue
		}
		previous := prior[vehicle.ID]
		if previous != nil && *previous == *vehicle.Fuel.SampleID {
			continue
		}
		samples = append(samples, dashboardFuelSample{VehicleID: vehicle.ID, SampleID: *vehicle.Fuel.SampleID, At: *vehicle.Fuel.MeasuredAt, ReceivedAt: vehicle.Fuel.ReceivedAt, Liters: vehicle.Fuel.Liters, Percent: vehicle.Fuel.Percent, Validity: vehicle.Fuel.Validity, CalibrationID: vehicle.Fuel.CalibrationID})
	}
	return samples
}

func (s *Server) dashboardNotifications(ctx context.Context, userID int64) ([]dashboardNotification, error) {
	rows, err := queryRows(ctx, s.DB, "SELECT id,title,body,target,read_at,created_at FROM notifications WHERE user_id=? ORDER BY id DESC LIMIT 200", userID)
	if err != nil {
		return nil, err
	}
	items := make([]dashboardNotification, 0, len(rows))
	for _, row := range rows {
		items = append(items, dashboardNotification{ID: dashboardString(row["id"], ""), Title: dashboardString(row["title"], "Notification"), Body: dashboardString(row["body"], ""), Target: dashboardString(row["target"], "/alerts"), ReadAt: dashboardText(row["read_at"]), CreatedAt: dashboardString(row["created_at"], "")})
	}
	return items, nil
}

func dashboardNotificationDelta(before, after []dashboardNotification) ([]dashboardNotification, []string) {
	prior := make(map[string]dashboardNotification, len(before))
	next := make(map[string]dashboardNotification, len(after))
	for _, item := range before {
		prior[item.ID] = item
	}
	upserts := make([]dashboardNotification, 0)
	for _, item := range after {
		next[item.ID] = item
		if old, ok := prior[item.ID]; !ok || !reflect.DeepEqual(old, item) {
			upserts = append(upserts, item)
		}
	}
	removed := make([]string, 0)
	for id := range prior {
		if _, ok := next[id]; !ok {
			removed = append(removed, id)
		}
	}
	sort.Strings(removed)
	return upserts, removed
}

func exactWebSocketOrigin(config *websocket.Config, request *http.Request) error {
	origin, err := url.Parse(request.Header.Get("Origin"))
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil {
		return errors.New("origin required")
	}
	if origin.Host != request.Host || (origin.Scheme != "https" && origin.Scheme != "http") {
		return errors.New("origin rejected")
	}
	config.Origin = origin
	return nil
}

func (s *Server) dashboardWebSocket(w http.ResponseWriter, r *http.Request) {
	s.wsHandshakes.Add(1)
	if r.Method != http.MethodGet {
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		fail(w, http.StatusForbidden, "origin_rejected")
		return
	}
	server := websocket.Server{
		Handshake: exactWebSocketOrigin,
		Handler: func(connection *websocket.Conn) {
			s.wsConnections.Add(1)
			defer s.wsConnections.Add(-1)
			s.dashboardSocket(connection)
		},
	}
	server.ServeHTTP(w, r)
}

func (s *Server) dashboardSocket(connection *websocket.Conn) {
	request := connection.Request()
	user, expires := s.sessionIdentity(request)
	if user == nil {
		s.wsAuthFailure.Add(1)
		_ = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "session_revoked", Reason: "authentication_required"})
		return
	}
	request = request.WithContext(context.WithValue(request.Context(), key{}, user))
	_ = connection.SetDeadline(time.Time{})
	changes := make(chan struct{}, 1)
	s.mu.Lock()
	s.subscribers[changes] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, changes)
		s.mu.Unlock()
	}()

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		var ignored any
		for {
			if err := websocket.JSON.Receive(connection, &ignored); err != nil {
				return
			}
		}
	}()

	vehicles, summary, asOf, err := s.dashboardSnapshotData(request.Context())
	if err != nil {
		_ = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "error", Reason: "dashboard_snapshot_failed"})
		return
	}
	notifications, err := s.dashboardNotifications(request.Context(), user.ID)
	if err != nil {
		_ = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "error", Reason: "notification_snapshot_failed"})
		return
	}
	cursor := s.wsCursor.Load()
	if cursor == 0 {
		cursor = s.wsCursor.Add(1)
	}
	if err = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "snapshot", Cursor: cursor, AsOf: asOf, Vehicles: vehicles, Summary: &summary, Notifications: notifications}); err != nil {
		return
	}

	expiry := time.NewTimer(time.Until(expires))
	defer expiry.Stop()
	heartbeat := time.NewTimer(25 * time.Second)
	defer heartbeat.Stop()
	freshness := time.NewTimer(time.Hour)
	if !freshness.Stop() {
		<-freshness.C
	}
	defer freshness.Stop()
	var freshnessChannel <-chan time.Time
	resetFreshness := func() {
		if !freshness.Stop() {
			select {
			case <-freshness.C:
			default:
			}
		}
		freshnessChannel = nil
		if next := dashboardNextFreshness(vehicles, time.Now().UTC()); !next.IsZero() {
			freshness.Reset(time.Until(next))
			freshnessChannel = freshness.C
		}
	}
	resetFreshness()
	for {
		select {
		case <-closed:
			return
		case <-expiry.C:
			_ = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "session_revoked", Cursor: cursor, Reason: "session_expired"})
			return
		case <-heartbeat.C:
			s.wsMessages.Add(1)
			if s.identity(request) == nil {
				_ = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "session_revoked", Cursor: cursor, Reason: "authentication_required"})
				return
			}
			if err = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "heartbeat", Cursor: cursor, AsOf: time.Now().UTC()}); err != nil {
				return
			}
			heartbeat.Reset(25 * time.Second)
		case <-freshnessChannel:
			freshnessChannel = nil
			select {
			case changes <- struct{}{}:
			default:
			}
		case <-changes:
			s.wsMessages.Add(1)
			if s.identity(request) == nil {
				_ = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "session_revoked", Cursor: cursor, Reason: "authentication_required"})
				return
			}
			next, nextSummary, nextAsOf, snapshotErr := s.dashboardSnapshotData(request.Context())
			if snapshotErr != nil {
				if err = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "error", Cursor: cursor, Reason: "dashboard_snapshot_failed"}); err != nil {
					return
				}
				continue
			}
			nextNotifications, notificationErr := s.dashboardNotifications(request.Context(), user.ID)
			if notificationErr != nil {
				if err = dashboardSocketSend(connection, dashboardSocketEnvelope{SchemaVersion: 1, Type: "error", Cursor: cursor, Reason: "notification_snapshot_failed"}); err != nil {
					return
				}
				continue
			}
			upserts, removed := dashboardDelta(vehicles, next)
			fuelSamples := dashboardFuelSamples(vehicles, next)
			notificationUpserts, notificationRemoved := dashboardNotificationDelta(notifications, nextNotifications)
			summaryChanged := !reflect.DeepEqual(summary, nextSummary)
			vehicles, summary = next, nextSummary
			notifications = nextNotifications
			resetFreshness()
			if len(upserts) == 0 && len(removed) == 0 && !summaryChanged && len(notificationUpserts) == 0 && len(notificationRemoved) == 0 {
				continue
			}
			cursor = s.wsCursor.Add(1)
			envelope := dashboardSocketEnvelope{SchemaVersion: 1, Type: "delta", Cursor: cursor, AsOf: nextAsOf, Upserts: upserts, Removed: removed, FuelSamples: fuelSamples, NotificationUpserts: notificationUpserts, NotificationRemoved: notificationRemoved}
			if summaryChanged {
				envelope.Summary = &nextSummary
			}
			if err = dashboardSocketSend(connection, envelope); err != nil {
				return
			}
			s.recordDashboardReplay(request.Context(), envelope)
		}
	}
}
