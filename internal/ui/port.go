package ui

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// freePortFunc frees addr so we can bind it. Overridable in tests.
var freePortFunc = freePort

// SetFreePortForTest replaces the port-reclaim helper. Pass nil to restore.
func SetFreePortForTest(fn func(addr string) error) {
	if fn == nil {
		freePortFunc = freePort
		return
	}
	freePortFunc = fn
}

// freePort kills a prior giffer listener on addr so this process can bind it.
// It refuses to kill unknown processes — Probe must confirm X-Giffer first.
func freePort(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse addr: %w", err)
	}
	if !Probe(httpURL(addr)) {
		return fmt.Errorf("port %s is in use by a non-giffer process; stop it manually or choose another --addr", port)
	}
	pids, err := listeningPIDs(port)
	if err != nil {
		return err
	}
	self := os.Getpid()
	var killed []int
	for _, pid := range pids {
		if pid <= 0 || pid == self {
			continue
		}
		if err := killPID(pid); err != nil {
			return fmt.Errorf("kill pid %d on port %s: %w", pid, port, err)
		}
		killed = append(killed, pid)
	}
	if len(killed) == 0 {
		return fmt.Errorf("port %s is in use but no listening process was found", port)
	}
	// Give the OS a moment to release the socket.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			_ = ln.Close()
			return nil
		}
		if !isAddrInUse(err) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("port %s still in use after killing %v", port, killed)
}

func listeningPIDs(port string) ([]int, error) {
	switch runtime.GOOS {
	case "windows":
		return listeningPIDsWindows(port)
	default:
		return listeningPIDsUnix(port)
	}
}

func listeningPIDsWindows(port string) ([]int, error) {
	out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat: %w", err)
	}
	needle := ":" + port
	seen := map[int]struct{}{}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if !strings.EqualFold(fields[3], "LISTENING") {
			continue
		}
		local := fields[1]
		if !portMatches(local, needle) {
			continue
		}
		pid, err := strconv.Atoi(fields[4])
		if err != nil {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	return pids, nil
}

func listeningPIDsUnix(port string) ([]int, error) {
	cmd := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN", "-t")
	out, err := cmd.Output()
	if err != nil {
		// lsof exits 1 when nothing matches.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 && len(bytes.TrimSpace(out)) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof: %w", err)
	}
	seen := map[int]struct{}{}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	return pids, nil
}

func portMatches(localAddr, needle string) bool {
	// localAddr is like 127.0.0.1:8765 or [::1]:8765 or 0.0.0.0:8765
	if strings.HasSuffix(localAddr, needle) {
		// Avoid :8765 matching :18765 — suffix after ':' must be exact port.
		i := strings.LastIndex(localAddr, ":")
		if i >= 0 && localAddr[i:] == needle {
			return true
		}
	}
	return false
}

func killPID(pid int) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
	} else {
		cmd = exec.Command("kill", "-TERM", strconv.Itoa(pid))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}
