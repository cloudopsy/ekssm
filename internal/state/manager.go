package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudopsy/ekssm/internal/logging"
)

type SessionState struct {
	PID            int    `json:"pid"`
	SessionID      string `json:"session_id"`
	ClusterName    string `json:"cluster_name"`
	InstanceID     string `json:"instance_id"`
	LocalPort      string `json:"local_port"`
	KubeconfigPath string `json:"kubeconfig_path"`
}

type SessionMap map[string]SessionState

type Manager struct {
	stateFilePath string
	mu            sync.Mutex // Protects access to the state file
}

func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	stateDir := filepath.Join(homeDir, ".ekssm")
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create state directory %s: %w", stateDir, err)
	}
	stateFilePath := filepath.Join(stateDir, "session.json")
	return &Manager{stateFilePath: stateFilePath}, nil
}

func (m *Manager) loadState() (SessionMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(SessionMap), nil
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", m.stateFilePath, err)
	}

	if len(data) == 0 {
		return make(SessionMap), nil
	}

	var sessions SessionMap
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state file %s: %w", m.stateFilePath, err)
	}
	return sessions, nil
}

func (m *Manager) saveState(sessions SessionMap) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}

	if err := os.WriteFile(m.stateFilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", m.stateFilePath, err)
	}
	logging.Debugf("Session state saved to %s", m.stateFilePath)
	return nil
}

func (m *Manager) AddSession(session SessionState) error {
	if session.SessionID == "" {
		return fmt.Errorf("cannot add session with empty SessionID")
	}
	sessions, err := m.loadState()
	if err != nil {
		return fmt.Errorf("failed to load state before adding session: %w", err)
	}
	sessions[session.SessionID] = session
	return m.saveState(sessions)
}

func (m *Manager) RemoveSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("cannot remove session with empty SessionID")
	}
	sessions, err := m.loadState()
	if err != nil {
		return fmt.Errorf("failed to load state before removing session: %w", err)
	}
	if _, exists := sessions[sessionID]; !exists {
		logging.Warnf("Attempted to remove non-existent session ID: %s", sessionID)
		return nil
	}
	delete(sessions, sessionID)
	return m.saveState(sessions)
}

func (m *Manager) GetSession(sessionID string) (*SessionState, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("cannot get session with empty SessionID")
	}
	sessions, err := m.loadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state before getting session: %w", err)
	}
	if session, exists := sessions[sessionID]; exists {
		return &session, nil
	}
	return nil, fmt.Errorf("session with ID '%s' not found", sessionID)
}

func (m *Manager) GetAllSessions() (SessionMap, error) {
	return m.loadState()
}

func (m *Manager) ClearAllSessions() error {
	return m.saveState(make(SessionMap))
}
