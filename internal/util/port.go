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

// FindAvailablePort finds an available local TCP port by listening on port 0.
// It returns the port number as a string.
func FindAvailablePort() (string, error) {
	// Listen on TCP port 0, which tells the OS to pick an available ephemeral port.
	// We listen on the loopback address only.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to listen on port 0: %w", err)
	}

	// Get the address (including the chosen port) that the listener is bound to.
	addr := listener.Addr().(*net.TCPAddr)

	// IMPORTANT: Close the listener immediately. We only needed it to find the port,
	// the actual service (SSM proxy) will bind to this port later.
	closeErr := listener.Close()
	if closeErr != nil {
		// Log this error, but finding the port was successful, so we might still proceed
		// Or return the error if closing is critical? Let's return it for safety.
		return "", fmt.Errorf("found port %d but failed to close listener: %w", addr.Port, closeErr)
	}

	port := addr.Port
	// Return the port number as a string
	return strconv.Itoa(port), nil
}
