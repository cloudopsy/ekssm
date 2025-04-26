package util

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/cloudopsy/ekssm/internal/logging"
)

func WaitForPort(port string, timeout time.Duration) error {
	address := fmt.Sprintf("localhost:%s", port)
	deadline := time.Now().Add(timeout)
	logging.Debugf("Waiting for port %s to become available (timeout: %s)...", port, timeout)

	var lastError error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			logging.Debugf("Port %s is now available", port)
			return nil
		}
		lastError = err
		logging.Debugf("Port %s not ready yet, retrying... Error: %v", port, err)
		time.Sleep(300 * time.Millisecond)
	}

	if lastError != nil {
		return fmt.Errorf("timed out waiting for local port %s to be ready. Last error: %v", port, lastError)
	}
	return fmt.Errorf("timed out waiting for local port %s to be ready", port)
}

func FindAvailablePort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to listen on port 0: %w", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	closeErr := listener.Close()
	if closeErr != nil {
		return "", fmt.Errorf("found port %d but failed to close listener: %w", addr.Port, closeErr)
	}

	return strconv.Itoa(addr.Port), nil
}
