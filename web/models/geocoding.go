package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Address struct {
	Status     string   `json:"status"`
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	Source     string   `json:"source"`
	ResultType string   `json:"result_type,omitempty"`
	Distance   *float64 `json:"distance_m,omitempty"`
	Cached     bool     `json:"cached"`
}

var geoClient = &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
var geoGuard = struct {
	sync.Mutex
	last time.Time
}{}

func geoKey() string {
	if key := strings.TrimSpace(os.Getenv("GEOAPIFY_API_KEY")); key != "" {
		return key
	}
	b, e := os.ReadFile(Env("GEOAPIFY_KEY_FILE", "/etc/pastrack/geoapify.key"))
	if e != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
func GeoReady() bool { return geoKey() != "" }
func decodeGeo(r io.Reader) (Address, error) {
	var data struct {
		Results []struct {
			Name      string   `json:"name"`
			Formatted string   `json:"formatted"`
			Line      string   `json:"address_line1"`
			Type      string   `json:"result_type"`
			Distance  *float64 `json:"distance"`
		} `json:"results"`
	}
	if e := json.NewDecoder(io.LimitReader(r, 128*1024)).Decode(&data); e != nil {
		return Address{}, fmt.Errorf("invalid response")
	}
	if len(data.Results) == 0 {
		return Address{Status: "not_found", Source: "Geoapify"}, nil
	}
	p := data.Results[0]
	name := p.Name
	if name == "" {
		name = p.Line
	}
	if name == "" {
		name = p.Formatted
	}
	if len(name) > 800 || len(p.Formatted) > 4000 || len(p.Type) > 40 {
		return Address{}, fmt.Errorf("invalid result")
	}
	return Address{Status: "ok", Name: name, Address: p.Formatted, Source: "Geoapify", ResultType: p.Type, Distance: p.Distance}, nil
}
func ResolveAddress(ctx context.Context, lat, lon float64) Address {
	places, e := Places(ctx)
	if e != nil {
		return Address{Status: "unavailable"}
	}
	var nearest *Place
	best := 1e9
	for _, p := range places {
		distance := coordinateDistance(lat, lon, p.Latitude, p.Longitude)
		if distance <= float64(p.Radius) && distance < best {
			copy := p
			nearest = &copy
			best = distance
		}
	}
	if nearest != nil {
		return Address{Status: "ok", Name: nearest.Name, Source: "Lokasi tersimpan", Distance: &best}
	}
	cell := fmt.Sprintf("%.5f/%.5f", lat, lon)
	var out Address
	var distance sql.NullFloat64
	e = DB.QueryRowContext(ctx, `SELECT name,address,result_type,distance_m FROM tracking_geocode_cache WHERE cell_key=? AND expires_at>UTC_TIMESTAMP()`, cell).Scan(&out.Name, &out.Address, &out.ResultType, &distance)
	if e == nil {
		out.Status = "ok"
		out.Source = "Geoapify"
		out.Cached = true
		if distance.Valid {
			out.Distance = &distance.Float64
		}
		if out.Address == "" && out.Name == "" {
			out.Status = "not_found"
		}
		return out
	}
	if e != sql.ErrNoRows {
		return Address{Status: "unavailable"}
	}
	key := geoKey()
	if key == "" {
		return Address{Status: "not_configured"}
	}
	geoGuard.Lock()
	if time.Since(geoGuard.last) < 1100*time.Millisecond {
		geoGuard.Unlock()
		return Address{Status: "rate_limited"}
	}
	geoGuard.last = time.Now()
	geoGuard.Unlock()
	limit, e := strconv.Atoi(Env("GEOAPIFY_DAILY_LIMIT", "500"))
	if e != nil || limit < 1 || limit > 100000 {
		limit = 500
	}
	_, e = DB.ExecContext(ctx, `INSERT IGNORE INTO tracking_geocode_usage(usage_day,requests) VALUES(UTC_DATE(),0)`)
	if e != nil {
		return Address{Status: "unavailable"}
	}
	r, e := DB.ExecContext(ctx, `UPDATE tracking_geocode_usage SET requests=requests+1 WHERE usage_day=UTC_DATE() AND requests<?`, limit)
	if e != nil {
		return Address{Status: "unavailable"}
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return Address{Status: "quota_exceeded"}
	}
	values := url.Values{"lat": {strconv.FormatFloat(lat, 'f', 7, 64)}, "lon": {strconv.FormatFloat(lon, 'f', 7, 64)}, "format": {"json"}, "lang": {"id"}, "limit": {"1"}, "apiKey": {key}}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.geoapify.com/v1/geocode/reverse?"+values.Encode(), nil)
	req.Header.Set("User-Agent", "Pastrack/1.0 (+https://toko.azlaundry.my.id)")
	resp, e := geoClient.Do(req)
	if e != nil {
		return Address{Status: "unavailable"}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return Address{Status: "credentials_rejected"}
	}
	if resp.StatusCode == 429 {
		return Address{Status: "rate_limited"}
	}
	if resp.StatusCode != 200 {
		return Address{Status: "unavailable"}
	}
	out, e = decodeGeo(resp.Body)
	if e != nil {
		return Address{Status: "unavailable"}
	}
	ttl := 30 * 24 * time.Hour
	if out.Status == "not_found" {
		ttl = 24 * time.Hour
	}
	_, _ = DB.ExecContext(ctx, `INSERT INTO tracking_geocode_cache(cell_key,name,address,result_type,distance_m,expires_at,updated_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE name=VALUES(name),address=VALUES(address),result_type=VALUES(result_type),distance_m=VALUES(distance_m),expires_at=VALUES(expires_at),updated_at=VALUES(updated_at)`, cell, out.Name, out.Address, out.ResultType, out.Distance, time.Now().UTC().Add(ttl))
	return out
}
