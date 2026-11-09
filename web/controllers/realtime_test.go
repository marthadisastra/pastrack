package controllers

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRealtimeHubCoalescesWithoutLosingResourceVersions(t *testing.T) {
	h := newRealtimeHub()
	changes, unsubscribe := h.subscribe()
	h.publish("positions")
	h.publish("master")
	h.publish("notifications")
	select {
	case <-changes:
	default:
		t.Fatal("subscriber not notified")
	}
	s := h.snapshot()
	if s.Version != 3 || s.Resources["positions"] != 1 || s.Resources["master"] != 2 || s.Resources["notifications"] != 3 {
		t.Fatal(s)
	}
	for len(changes) > 0 {
		<-changes
	}
	unsubscribe()
	h.publish("positions")
	select {
	case <-changes:
		t.Fatal("unsubscribed client notified")
	default:
	}
}
func TestRealtimeSocketPrivateAndEventDriven(t *testing.T) {
	dir, e := os.MkdirTemp("/tmp", "pt-live-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "live.sock")
	conn, e := openRealtimeSocket(path)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("socket permissions")
	}
	if other, e := openRealtimeSocket(path); e == nil {
		other.Close()
		t.Fatal("active socket replaced")
	}
	events := make(chan string, 1)
	go serveRealtimeSocket(conn, func(kind string) { events <- kind })
	sender, e := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: path, Net: "unixgram"})
	if e != nil {
		t.Fatal(e)
	}
	defer sender.Close()
	sender.Write([]byte("positions"))
	select {
	case kind := <-events:
		if kind != "positions" {
			t.Fatal(kind)
		}
	case <-time.After(time.Second):
		t.Fatal("commit event not delivered")
	}
}
func TestRealtimeStreamSendsChangesImmediately(t *testing.T) {
	old := liveHub
	liveHub = newRealtimeHub()
	defer func() { liveHub = old }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), userContextKey{}, &User{ID: 1, Active: true}))
		realtimeStream(w, r)
	}))
	defer server.Close()
	client := &http.Client{Timeout: 2 * time.Second}
	response, e := client.Get(server.URL)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	read := func() realtimeEvent {
		t.Helper()
		for {
			line, e := reader.ReadString('\n')
			if e != nil {
				t.Fatal(e)
			}
			if strings.HasPrefix(line, "data: ") {
				var event realtimeEvent
				if e = json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data: "))), &event); e != nil {
					t.Fatal(e)
				}
				return event
			}
		}
	}
	first := read()
	publishRealtime("positions")
	next := read()
	if next.Instance != first.Instance || next.Version != first.Version+1 {
		t.Fatal(first, next)
	}
}
