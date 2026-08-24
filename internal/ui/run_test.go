package ui_test

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlienFacepalm/giffer/internal/ui"
)

func TestProbeDetectsGiffer(t *testing.T) {
	srv := ui.New(ui.Options{UploadDir: t.TempDir()})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = http.Serve(ln, srv.Handler()) }()

	url := "http://" + ln.Addr().String()
	waitProbe(t, url)

	if !ui.Probe(url) {
		t.Fatal("Probe should recognize giffer UI")
	}
	if ui.Probe("http://127.0.0.1:1") {
		t.Fatal("Probe should fail for unreachable addr")
	}
}

func TestRunReclaimsExistingPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	held := true
	t.Cleanup(func() {
		if held {
			_ = ln.Close()
		}
	})

	freed := make(chan struct{}, 1)
	ui.SetFreePortForTest(func(a string) error {
		if a != addr {
			t.Fatalf("freePort addr=%q, want %s", a, addr)
		}
		if err := ln.Close(); err != nil {
			return err
		}
		held = false
		freed <- struct{}{}
		return nil
	})
	t.Cleanup(func() { ui.SetFreePortForTest(nil) })

	opened := make(chan string, 1)
	blockWindow := make(chan struct{})
	ui.SetOpenWindowForTest(func(u string) error {
		opened <- u
		<-blockWindow
		return nil
	})
	t.Cleanup(func() {
		close(blockWindow)
		ui.SetOpenWindowForTest(nil)
	})

	var out syncBuffer
	done := make(chan error, 1)
	go func() {
		done <- ui.Run(ui.Options{Addr: addr, UploadDir: t.TempDir()}, &out)
	}()

	select {
	case <-freed:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for port reclaim")
	}

	url := "http://" + addr
	waitProbe(t, url)

	got := out.String()
	if !strings.Contains(got, "reclaiming") {
		t.Fatalf("stdout=%q, want reclaiming notice", got)
	}
	if !strings.Contains(got, "listening on") {
		t.Fatalf("stdout=%q, want listening notice", got)
	}

	select {
	case u := <-opened:
		if u != url {
			t.Fatalf("opened %q, want %s", u, url)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for window open")
	}

	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "closed") {
			t.Fatalf("Run exited early: %v", err)
		}
	default:
		// still serving — expected
	}
}

func TestRunOpensWindowOnListen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	opened := make(chan string, 1)
	blockWindow := make(chan struct{})
	ui.SetOpenWindowForTest(func(u string) error {
		opened <- u
		<-blockWindow
		return nil
	})
	t.Cleanup(func() {
		close(blockWindow)
		ui.SetOpenWindowForTest(nil)
	})

	done := make(chan error, 1)
	go func() {
		done <- ui.Run(ui.Options{Addr: addr, UploadDir: t.TempDir()}, ioDiscard{})
	}()

	url := "http://" + addr
	waitProbe(t, url)

	select {
	case u := <-opened:
		if u != url {
			t.Fatalf("opened %q, want %s", u, url)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for window open")
	}

	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "closed") {
			t.Fatalf("Run exited early: %v", err)
		}
	default:
		// still serving — expected
	}
}

func waitProbe(t *testing.T, url string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if ui.Probe(url) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("Probe never succeeded for %s", url)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}
