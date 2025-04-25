package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudopsy/ekssm/internal/logging"
)

type SessionState struct {
	PID         int    `json:"pid"`
	SessionID   string `json:"session_id"`
	ClusterName string `json:"cluster_name"`
	InstanceID  string `json:"instance_id"`
	LocalPort   string `json:"local_port"`
}

func stateFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ekssm", "session.json"), nil
}

func ReadState() (*SessionState, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logging.Debug("State file does not exist, assuming no active session.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", path, err)
	}

	if len(data) == 0 {
		logging.Debug("State file is empty, assuming no active session.")
		return nil, nil
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		logging.Errorf("State file %s contains invalid JSON: %v", path, err)
		return nil, fmt.Errorf("failed to parse state file %s (corrupted?): %w", path, err)
	}

	if state.PID <= 0 {
		logging.Warnf("State file contains invalid PID (%d). Clearing state.", state.PID)
		_ = ClearState()
		return nil, nil
	}

	logging.Debugf("Read active session state: PID=%d, SessionID=%s", state.PID, state.SessionID)
	return &state, nil
}

func WriteState(state *SessionState) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create state directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", path, err)
	}

	logging.Debugf("Session state written to %s", path)
	return nil
}

func ClearState() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logging.Debug("State file already removed.")
			return nil
		}
		return fmt.Errorf("failed to remove state file %s: %w", path, err)
	}

	logging.Debugf("Session state file %s removed.", path)
	return nil
}
