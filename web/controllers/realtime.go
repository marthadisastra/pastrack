package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type realtimeEvent struct {
	Instance  string            `json:"instance"`
	Version   uint64            `json:"version"`
	At        time.Time         `json:"at"`
	Resources map[string]uint64 `json:"resources"`
}
type realtimeHub struct {
	sync.Mutex
	instance    string
	version     uint64
	updated     time.Time
	resources   map[string]uint64
	subscribers map[chan []byte]struct{}
}

func newRealtimeHub() *realtimeHub {
	return &realtimeHub{instance: requestID(), updated: time.Now().UTC(), resources: map[string]uint64{"positions": 0, "master": 0, "notifications": 0, "users": 0, "places": 0}, subscribers: map[chan []byte]struct{}{}}
}

var liveHub = newRealtimeHub()

func (h *realtimeHub) publish(resource string) {
	h.Lock()
	defer h.Unlock()
	if _, ok := h.resources[resource]; !ok {
		return
	}
	h.version++
	h.resources[resource] = h.version
	h.updated = time.Now().UTC()
	for subscriber := range h.subscribers {
		select {
		case subscriber <- []byte("positions"):
		default:
		}
	}
}
func (h *realtimeHub) publishEntity(payload []byte) {
	h.Lock()
	defer h.Unlock()
	for subscriber := range h.subscribers {
		select {
		case subscriber <- payload:
		default:
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- []byte("positions"):
			default:
			}
		}
	}
}
func (h *realtimeHub) subscribe() (chan []byte, func()) {
	h.Lock()
	defer h.Unlock()
	c := make(chan []byte, 256)
	h.subscribers[c] = struct{}{}
	return c, func() { h.Lock(); delete(h.subscribers, c); h.Unlock() }
}
func (h *realtimeHub) snapshot() realtimeEvent {
	h.Lock()
	defer h.Unlock()
	event := realtimeEvent{Instance: h.instance, Version: h.version, At: h.updated, Resources: map[string]uint64{}}
	for k, v := range h.resources {
		event.Resources[k] = v
	}
	return event
}
func publishRealtime(resource string) { liveHub.publish(resource) }
func openRealtimeSocket(path string) (*net.UnixConn, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	if info, e := os.Lstat(path); e == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("realtime path is not a socket")
		}
		probe, e := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: path, Net: "unixgram"})
		if e == nil {
			probe.Close()
			return nil, fmt.Errorf("realtime socket is already active")
		}
		if e = os.Remove(path); e != nil {
			return nil, e
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	conn, e := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if e != nil {
		return nil, e
	}
	if e = os.Chmod(path, 0600); e != nil {
		conn.Close()
		return nil, e
	}
	return conn, nil
}
func startRealtime() {
	path := env("REALTIME_SOCKET_PATH", "/tmp/pastrack/live.sock")
	if path == "off" {
		return
	}
	conn, e := openRealtimeSocket(path)
	if e != nil {
		log.Fatalf("Realtime socket: %v", e)
	}
	go serveRealtimeSocket(conn, publishRealtime)
}
func serveRealtimeSocket(conn *net.UnixConn, publish func(string)) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	for {
		n, _, e := conn.ReadFromUnix(buffer)
		if e != nil {
			if !errors.Is(e, net.ErrClosed) {
				log.Printf("Realtime socket read: %v", e)
			}
			return
		}
		if string(buffer[:n]) == "positions" {
			publish("positions")
			continue
		}
		var change struct {
			Kind     string `json:"kind"`
			IMEI     string `json:"imei"`
			SensorID string `json:"sensor_id"`
		}
		if json.Unmarshal(buffer[:n], &change) == nil && (change.Kind == "gps" || change.Kind == "fuel" || change.Kind == "report") && change.IMEI != "" {
			liveHub.publishEntity(append([]byte(nil), buffer[:n]...))
		}
	}

}
func realtimeStream(w http.ResponseWriter, r *http.Request) {
	if !authenticated(r) {
		http.Error(w, "Sesi berakhir", 401)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak tersedia", 503)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	changes, unsubscribe := liveHub.subscribe()
	defer unsubscribe()
	send := func() {
		event := liveHub.snapshot()
		b, _ := json.Marshal(event)
		fmt.Fprintf(w, "event: positions\ndata: %s\n\n", b)
		flusher.Flush()
	}
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(40 * time.Second))
	fmt.Fprint(w, "retry: 3000\n\n")
	send()
	// Heartbeats maintain the connection; they never query positions or trigger UI refresh.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case payload := <-changes:
			if string(payload) == "positions" {
				send()
			} else {
				fmt.Fprintf(w, "event: entity\ndata: %s\n\n", payload)
				flusher.Flush()
			}
		case <-heartbeat.C:
			if resolveUser(r) == nil {
				fmt.Fprint(w, "event: session\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			_ = controller.SetWriteDeadline(time.Now().Add(40 * time.Second))
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
func RealtimeStream(w http.ResponseWriter, r *http.Request) { realtimeStream(w, r) }
