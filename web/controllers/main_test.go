package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSessionValidation(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if authenticated(r) {
		t.Fatal("missing session authenticated")
	}
	r.AddCookie(&http.Cookie{Name: "pastrack_session", Value: "legacy-expiry.forged-signature"})
	if authenticated(r) {
		t.Fatal("legacy cookie was accepted")
	}
	r = r.WithContext(context.WithValue(r.Context(), userContextKey{}, &User{ID: 1, Username: "admin", Role: "admin", Active: true}))
	if !authenticated(r) {
		t.Fatal("resolved session was rejected")
	}
	if len(tokenHash("session-token")) != 64 || tokenHash("session-token") == tokenHash("different-token") {
		t.Fatal("session hashing failed")
	}
}
func TestQueryBounds(t *testing.T) {
	for _, s := range []string{"-1", "100000000000000000000", "1 OR 1=1", "xyz"} {
		if integer(s, 1, 1, 10000) != 1 {
			t.Fatalf("unsafe input accepted %q", s)
		}
	}
}
func TestLoginRateLimit(t *testing.T) {
	ip := "test-rate-limit"
	for i := 0; i < 15; i++ {
		if !allowLogin(ip) {
			t.Fatalf("attempt %d denied too early", i)
		}
	}
	if allowLogin(ip) {
		t.Fatal("rate limit not enforced")
	}
}
func TestHeaders(t *testing.T) {
	r := httptest.NewRecorder()
	security(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })).ServeHTTP(r, httptest.NewRequest("GET", "/", nil))
	if r.Header().Get("X-Frame-Options") != "DENY" || !strings.Contains(r.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Fatal("missing security headers")
	}
}
