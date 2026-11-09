package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func (s *Server) remoteOutput(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	p := strings.Split(path, "/")
	if len(p) != 3 || p[0] != "vehicles" || p[2] != "remote-output" {
		return false
	}
	id, e := parseID(p[1])
	if e != nil {
		fail(w, 400, "invalid_vehicle")
		return true
	}
	if r.Method != "GET" && r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if r.Method == "POST" && u.Role != "admin" {
		fail(w, 403, "permission_denied")
		return true
	}
	socket := os.Getenv("REMOTE_OUTPUT_SOCKET")
	if socket == "" || socket == "off" {
		if r.Method == "GET" {
			respond(w, 200, map[string]any{"data": map[string]any{"configured": false, "connected": false, "confirmed_active": nil, "reason": "Layanan output relay belum dikonfigurasi"}})
		} else {
			fail(w, 409, "Layanan output relay belum dikonfigurasi")
		}
		return true
	}
	var payload []byte
	if r.Method == "POST" {
		var input struct {
			Active *bool  `json:"active"`
			Key    string `json:"request_key"`
		}
		if body(w, r, &input) != nil || input.Active == nil || len(input.Key) != 64 {
			fail(w, 400, "invalid_output_request")
			return true
		}
		payload, _ = json.Marshal(map[string]any{"active": input.Active, "request_key": input.Key, "actor_id": u.ID})
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, r.Method, "http://receiver/output?vehicle_id="+strconv.FormatInt(id, 10), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	response, e := client.Do(req)
	if e != nil {
		fail(w, 503, "Layanan receiver output belum tersedia")
		return true
	}
	defer response.Body.Close()
	data, e := io.ReadAll(io.LimitReader(response.Body, 65536))
	if e != nil {
		fail(w, 503, "output_response_unavailable")
		return true
	}
	var result map[string]any
	if json.Unmarshal(data, &result) != nil {
		fail(w, 503, "invalid_output_response")
		return true
	}
	if response.StatusCode >= 400 {
		message, _ := result["error"].(string)
		fail(w, response.StatusCode, message)
		return true
	}
	if r.Method == "POST" {
		s.Notify()
	}
	respond(w, response.StatusCode, result)
	return true
}
