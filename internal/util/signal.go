// Package util provides internal utility functions for file operations, networking,
// and kubeconfig management specific to ekssm.
package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudopsy/ekssm/internal/logging"
)

// SignalContext returns a context that is canceled when an interrupt
// or termination signal is received. It also returns a function that should
// be called to clean up the signal handling.
func SignalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-signalCh:
			logging.Debugf("Received signal: %v, canceling context", sig)
			cancel()
		case <-ctx.Done():
			// Context was canceled elsewhere
		}
	}()

	// Return a cleanup function that stops signal notifications
	return ctx, func() {
		signal.Stop(signalCh)
		cancel()
	}
}

// HandleSignalCustom sets up a signal handler with a custom cleanup function
// that will be executed when an interrupt or termination signal is received.
func HandleSignalCustom(cleanup func()) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalCh
		logging.Infof("Received signal: %v. Cleaning up...", sig)
		
		// Execute the cleanup function
		cleanup()
		
		// Stop handling signals
		signal.Stop(signalCh)
		
		// Exit with a non-zero code to indicate termination by signal
		os.Exit(1)
	}()
}