package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRealtimePublisherWakesOnCommitAndStops(t *testing.T) {
	dir, e := os.MkdirTemp("/tmp", "pt-live-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "live.sock")
	conn, e := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	old := realtimeChanges
	realtimeChanges = make(chan []byte, 2048)
	defer func() { realtimeChanges = old }()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { publishChanges(ctx, nil, path); close(done) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("publisher failed to stop")
		}
	}()
	read := func() {
		t.Helper()
		conn.SetReadDeadline(time.Now().Add(time.Second))
		buf := make([]byte, 128)
		n, _, err := conn.ReadFromUnix(buf)
		if err != nil || string(buf[:n]) != "positions" {
			t.Fatalf("bad invalidation %q %v", buf[:n], err)
		}
	}
	read()
	signalRealtime()
	read()
}
func TestEnqueueSignalsWorkerAfterDurableWrite(t *testing.T) {
	old := spoolReady
	spoolReady = make(chan struct{}, 1)
	defer func() { spoolReady = old }()
	if e := enqueue(t.TempDir(), Record{Payload: "$", Received: time.Now()}); e != nil {
		t.Fatal(e)
	}
	select {
	case <-spoolReady:
	default:
		t.Fatal("worker not woken by enqueue")
	}
}
