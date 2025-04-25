// Package proxy provides functionality for establishing secure connections
// to remote services through SSM port forwarding.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/cloudopsy/ekssm/internal/logging"
	"github.com/cloudopsy/ekssm/internal/util"
	awsclient "github.com/cloudopsy/ekssm/pkg/aws"
)

// SSMProxy manages an SSM port forwarding session.
type SSMProxy struct {
	InstanceID string
	LocalPort  string
	RemoteHost string
	RemotePort string
	cmd        *exec.Cmd
	SessionID  string
	ctx        context.Context
	client     *awsclient.Client
}

// NewSSMProxy creates a new SSM proxy configured with the specified parameters.
func NewSSMProxy(instanceID, localPort, remoteHost, remotePort string) *SSMProxy {
	if remotePort == "" {
		remotePort = "443"
	}
	return &SSMProxy{
		InstanceID: instanceID,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		ctx:        context.Background(),
	}
}

// StartBackground starts the SSM proxy process in the background and returns its PID.
func (p *SSMProxy) StartBackground() (int, error) {
	if p.InstanceID == "" {
		return -1, fmt.Errorf("instanceID is required")
	}
	if p.LocalPort == "" {
		return -1, fmt.Errorf("localPort is required")
	}
	if p.RemoteHost == "" {
		return -1, fmt.Errorf("remoteHost (EKS endpoint) is required")
	}
	if p.RemotePort == "" {
		return -1, fmt.Errorf("remotePort is required")
	}

	logging.Debugf("Starting SSM port forwarding to remote host %s:%s via instance %s on local port %s",
		p.RemoteHost, p.RemotePort, p.InstanceID, p.LocalPort)

	var err error
	p.client, err = awsclient.NewClient(p.ctx)
	if err != nil {
		logging.Errorf("Failed to create AWS client: %v", err)
		return -1, fmt.Errorf("failed to create AWS client: %w", err)
	}

	documentName := "AWS-StartPortForwardingSessionToRemoteHost"
	parameters := map[string][]string{
		"localPortNumber": {p.LocalPort},
		"host":            {p.RemoteHost},
		"portNumber":      {p.RemotePort},
	}

	result, err := p.client.SSM.StartSession(p.ctx, &ssm.StartSessionInput{
		Target:       aws.String(p.InstanceID),
		DocumentName: aws.String(documentName),
		Parameters:   parameters,
	})
	if err != nil {
		logging.Errorf("Failed to start SSM session via API: %v", err)
		return -1, fmt.Errorf("failed to start SSM session: %w", err)
	}

	p.SessionID = *result.SessionId

	sessionInput, err := createSessionInput(result, p.InstanceID, parameters, documentName)
	if err != nil {
		logging.Errorf("Failed to create session-manager-plugin input JSON: %v", err)
		return -1, fmt.Errorf("failed to create session input: %w", err)
	}

	pluginPath := getPluginPath()
	var stderrBuf bytes.Buffer
	region := p.client.Region
	if region == "" {
		logging.Errorf("AWS region not found in AWS client configuration")
		return -1, fmt.Errorf("AWS region not set for session-manager-plugin invocation")
	}
	args := []string{sessionInput, region, "StartSession"}
	p.cmd = exec.Command(pluginPath, args...)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = &stderrBuf

	err = p.cmd.Start()
	if err != nil {
		if errMsg := stderrBuf.String(); errMsg != "" {
			logging.Errorf("session-manager-plugin stderr on start: %s", errMsg)
		}
		logging.Errorf("Failed to start session-manager-plugin process: %v", err)
		return -1, fmt.Errorf("failed to start session-manager-plugin: %w", err)
	}

	timeout := 30 * time.Second
	if err := util.WaitForPort(p.LocalPort, timeout); err != nil {
		errMsg := stderrBuf.String()
		if errMsg != "" {
			logging.Errorf("Session-manager-plugin stderr during port wait: %s", errMsg)
		}
		logging.Errorf("Timed out waiting for local port %s: %v", p.LocalPort, err)
		_ = p.Stop()
		return -1, fmt.Errorf("timed out waiting for port %s: %w - check plugin logs, permissions, network, and SSM agent status", p.LocalPort, err)
	}

	return p.cmd.Process.Pid, nil
}

// Stop terminates the SSM proxy session and cleans up resources.
func (p *SSMProxy) Stop() error {
	var firstErr error

	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Signal(os.Interrupt); err != nil {
			logging.Errorf("Failed to send interrupt signal to session-manager-plugin process: %v", err)
			firstErr = fmt.Errorf("failed to send interrupt signal to plugin process: %w", err)
		}
		_ = p.cmd.Wait()
	}

	if p.client != nil && p.SessionID != "" {
		_, err := p.client.SSM.TerminateSession(p.ctx, &ssm.TerminateSessionInput{
			SessionId: aws.String(p.SessionID),
		})
		if err != nil {
			logging.Warnf("Failed to terminate SSM session %s via API: %v", p.SessionID, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to terminate SSM session API call: %w", err)
			}
		}
	}

	p.cmd = nil
	p.SessionID = ""

	return firstErr
}

type pluginSessionInput struct {
	Target       string              `json:"Target"`
	DocumentName string              `json:"DocumentName"`
	Parameters   map[string][]string `json:"Parameters"`
	SessionId    string              `json:"SessionId"`
	StreamUrl    string              `json:"StreamUrl"`
	TokenValue   string              `json:"TokenValue"`
}

func createSessionInput(session *ssm.StartSessionOutput, instanceID string, parameters map[string][]string, documentName string) (string, error) {
	input := pluginSessionInput{
		Target:       instanceID,
		DocumentName: documentName,
		Parameters:   parameters,
		SessionId:    *session.SessionId,
		StreamUrl:    *session.StreamUrl,
		TokenValue:   *session.TokenValue,
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session input to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

func getPluginPath() string {
	switch runtime.GOOS {
	case "windows":
		return "session-manager-plugin.exe"
	default:
		return "session-manager-plugin"
	}
}
