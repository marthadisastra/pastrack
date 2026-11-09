package controllers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type pushTransport func(*http.Request) (*http.Response, error)

func (f pushTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestPushV1PayloadAndRevokedToken(t *testing.T) {
	old := pushHTTP
	defer func() { pushHTTP = old }()
	pushHTTP = &http.Client{Transport: pushTransport(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		s := string(b)
		if r.URL.Host != "fcm.googleapis.com" || !strings.Contains(s, `"data"`) || strings.Contains(s, `"notification"`) || !strings.Contains(s, `"url":"/#notifications"`) {
			t.Fatalf("Unexpected FCM payload %s", s)
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(`{"error":{"details":[{"errorCode":"UNREGISTERED"}]}}`)), Header: make(http.Header)}, nil
	})}
	status, invalid, e := pushSend(context.Background(), "test-token", "Test", "Body", "/#notifications")
	if status != 404 || !invalid || e == nil {
		t.Fatalf("invalid token not classified %v %v %v", status, invalid, e)
	}
}
func TestNotificationRejectsUnsafeTargets(t *testing.T) {
	for _, target := range []string{"https://example.com", "//example.com", "javascript:alert(1)", "/login"} {
		if createNotification(context.Background(), 1, "Test", "body", target) == nil {
			t.Fatal("unsafe target accepted")
		}
	}
}
func TestRealtimeRejectsAnonymous(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/realtime", nil)
	w := httptest.NewRecorder()
	realtimeStream(w, r)
	if w.Code != 401 {
		t.Fatalf("Anonymous stream status %d", w.Code)
	}
}
