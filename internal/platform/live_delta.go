package platform

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
)

type liveRows struct {
	Rows  map[string]map[string]any `json:"rows"`
	Order []string                  `json:"order"`
	Page  map[string]any            `json:"page,omitempty"`
}
type liveSectionPatch struct {
	Upserts map[string]map[string]any `json:"upserts"`
	Removed []string                  `json:"removed"`
	Order   []string                  `json:"order,omitempty"`
	Page    map[string]any            `json:"page,omitempty"`
}

func diffLive(old, next liveRows) liveSectionPatch {
	p := liveSectionPatch{Upserts: map[string]map[string]any{}, Removed: []string{}}
	if !reflect.DeepEqual(old.Page, next.Page) {
		p.Page = next.Page
	}
	if !reflect.DeepEqual(old.Order, next.Order) {
		p.Order = next.Order
	}
	for id, row := range next.Rows {
		changed := map[string]any{}
		for field, value := range row {
			if prior, ok := old.Rows[id][field]; !ok || !reflect.DeepEqual(prior, value) {
				changed[field] = value
			}
		}
		for field := range old.Rows[id] {
			if _, ok := row[field]; !ok {
				changed[field] = nil
			}
		}
		if len(changed) > 0 {
			p.Upserts[id] = changed
		}
	}
	for id := range old.Rows {
		if _, ok := next.Rows[id]; !ok {
			p.Removed = append(p.Removed, id)
		}
	}
	return p
}
func (s *Server) liveSnapshot(r *http.Request, u *Identity, endpoint string) (liveRows, error) {
	v, e := url.Parse(endpoint)
	if e != nil || v.IsAbs() || v.Host != "" || strings.Contains(v.Path, "..") || len(endpoint) > 2048 {
		return liveRows{}, fmt.Errorf("invalid live endpoint")
	}
	// Only list/read routes may be subscribed to; route handlers enforce the normal role scope.
	if _, ok := resources[v.Path]; !ok && v.Path != "fleet/live" && v.Path != "fleet/overview" && v.Path != "fleet/activity" && !strings.HasPrefix(v.Path, "vehicles/") {
		return liveRows{}, fmt.Errorf("unsupported live endpoint")
	}
	req := r.Clone(r.Context())
	req.Method = "GET"
	req.URL = &url.URL{Path: "/api/v1/" + v.Path, RawQuery: v.RawQuery}
	rec := httptest.NewRecorder()
	s.api(rec, req, u)
	if rec.Code != 200 {
		return liveRows{}, fmt.Errorf("live endpoint status %d", rec.Code)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
		Page map[string]any  `json:"page"`
	}
	if e = json.Unmarshal(rec.Body.Bytes(), &envelope); e != nil {
		return liveRows{}, e
	}
	rows := []map[string]any{}
	if len(envelope.Data) > 0 && envelope.Data[0] == '[' {
		e = json.Unmarshal(envelope.Data, &rows)
	} else {
		var row map[string]any
		e = json.Unmarshal(envelope.Data, &row)
		if row != nil {
			rows = append(rows, row)
		}
	}
	if e != nil {
		return liveRows{}, e
	}
	out := liveRows{Rows: map[string]map[string]any{}, Order: []string{}, Page: envelope.Page}
	for i, row := range rows {
		id := fmt.Sprint(row["id"])
		if row["id"] == nil {
			id = fmt.Sprint(i)
		}
		out.Rows[id] = row
		out.Order = append(out.Order, id)
	}
	return out, nil
}
