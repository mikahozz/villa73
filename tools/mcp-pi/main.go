package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	serverName               = "villa73-raspberry-pi"
	serverVersion            = "0.2.0"
	defaultComposeLogsTail   = 200
	maxComposeLogsTail       = 1000
	maxRemoteCommandOutput   = 512 * 1024
	sshCommandTimeoutSeconds = 45
)

type server struct {
	reader *bufio.Reader
	writer *bufio.Writer
	config sshConfig
}

type sshConfig struct {
	target          string
	port            string
	identityFile    string
	certificateFile string
	knownHostsFile  string
	allowedProjects map[string]string
}

type requestEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type responseEnvelope struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type toolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type composeLogsArgs struct {
	Target      string   `json:"target"`
	Services    []string `json:"services"`
	ComposeFile string   `json:"composeFile"`
	Tail        int      `json:"tail"`
	Since       string   `json:"since"`
	Timestamps  bool     `json:"timestamps"`
}

type serviceRequestArgs struct {
	Target         string            `json:"target"`
	Service        string            `json:"service"`
	ContainerPort  int               `json:"containerPort"`
	Path           string            `json:"path"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	Scheme         string            `json:"scheme"`
	TimeoutSeconds int               `json:"timeoutSeconds"`
}

type toolResult struct {
	Content []toolTextContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}

type toolTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	s := &server{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
		config: loadSSHConfig(),
	}

	for {
		payload, err := readFrame(s.reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			writeFatalProtocolError(s.writer, err)
			return
		}

		var req requestEnvelope
		if err := json.Unmarshal(payload, &req); err != nil {
			_ = writeFrame(s.writer, responseEnvelope{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "invalid JSON payload", Data: err.Error()},
			})
			continue
		}

		resp := s.handleRequest(req)
		if resp == nil {
			continue
		}
		if err := writeFrame(s.writer, resp); err != nil {
			return
		}
	}
}

func loadSSHConfig() sshConfig {
	target := strings.TrimSpace(os.Getenv("MCP_PI_SSH_TARGET"))
	allowedProjects := map[string]string{}
	if dir := strings.TrimSpace(os.Getenv("MCP_PI_ALLOWED_PROJECT_DIR1")); dir != "" {
		allowedProjects["dir1"] = dir
	}
	if dir := strings.TrimSpace(os.Getenv("MCP_PI_ALLOWED_PROJECT_DIR2")); dir != "" {
		allowedProjects["dir2"] = dir
	}

	return sshConfig{
		target:          target,
		port:            strings.TrimSpace(os.Getenv("MCP_PI_SSH_PORT")),
		identityFile:    strings.TrimSpace(os.Getenv("MCP_PI_SSH_IDENTITY_FILE")),
		certificateFile: strings.TrimSpace(os.Getenv("MCP_PI_SSH_CERTIFICATE_FILE")),
		knownHostsFile:  strings.TrimSpace(os.Getenv("MCP_PI_SSH_KNOWN_HOSTS_FILE")),
		allowedProjects: allowedProjects,
	}
}

func (s *server) handleRequest(req requestEnvelope) *responseEnvelope {
	id := decodeID(req.ID)

	switch req.Method {
	case "initialize":
		return &responseEnvelope{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]string{
					"name":    serverName,
					"version": serverVersion,
				},
				"instructions": "Runs on the laptop and uses SSH to the Raspberry Pi. Callers must choose one configured target name, dir1 or dir2, which map to preconfigured Docker Compose project directories.",
			},
		}
	case "notifications/initialized":
		return nil
	case "ping":
		return &responseEnvelope{JSONRPC: "2.0", ID: id, Result: map[string]interface{}{}}
	case "tools/list":
		return &responseEnvelope{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]interface{}{
				"tools": []toolDefinition{
					composeLogsTool(),
					serviceRequestTool(),
				},
			},
		}
	case "tools/call":
		var params toolsCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(id, -32602, "invalid tools/call params", err.Error())
		}
		result, err := s.callTool(params)
		if err != nil {
			return &responseEnvelope{
				JSONRPC: "2.0",
				ID:      id,
				Result: toolResult{
					Content: []toolTextContent{{Type: "text", Text: err.Error()}},
					IsError: true,
				},
			}
		}
		return &responseEnvelope{JSONRPC: "2.0", ID: id, Result: result}
	default:
		return errorResponse(id, -32601, "method not found", req.Method)
	}
}

func composeLogsTool() toolDefinition {
	return toolDefinition{
		Name:        "docker_compose_logs",
		Description: "Read logs from a Docker Compose solution on the Raspberry Pi by directory, with optional service filtering.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]string{
					"type":        "string",
					"description": "Configured project selector. Must be dir1 or dir2.",
				},
				"services": map[string]interface{}{
					"type":        "array",
					"description": "Optional list of services to filter logs for.",
					"items":       map[string]string{"type": "string"},
				},
				"composeFile": map[string]string{
					"type":        "string",
					"description": "Optional compose file path, absolute or relative to projectDir.",
				},
				"tail": map[string]interface{}{
					"type":        "integer",
					"description": "How many log lines to return. Defaults to 200.",
					"default":     200,
					"minimum":     1,
				},
				"since": map[string]string{
					"type":        "string",
					"description": "Optional Docker logs since value such as 15m, 2h, or an RFC3339 timestamp.",
				},
				"timestamps": map[string]interface{}{
					"type":        "boolean",
					"description": "Include Docker timestamps in the output.",
					"default":     false,
				},
			},
			"required": []string{"target"},
		},
	}
}

func serviceRequestTool() toolDefinition {
	return toolDefinition{
		Name:        "compose_service_api_request",
		Description: "Execute an HTTP request from the Raspberry Pi host to a service declared by a Docker Compose project. The target port must be an exposed container port from that compose service.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]string{
					"type":        "string",
					"description": "Configured project selector. Must be dir1 or dir2.",
				},
				"service": map[string]string{
					"type":        "string",
					"description": "Compose service name.",
				},
				"containerPort": map[string]interface{}{
					"type":        "integer",
					"description": "Container port exposed by the compose service. The server resolves the published host port using docker compose port.",
					"minimum":     1,
				},
				"path": map[string]string{
					"type":        "string",
					"description": "Request path, starting with /. Defaults to /.",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method. Defaults to GET.",
					"default":     "GET",
				},
				"headers": map[string]interface{}{
					"type":                 "object",
					"description":          "Optional HTTP headers.",
					"additionalProperties": map[string]string{"type": "string"},
				},
				"body": map[string]string{
					"type":        "string",
					"description": "Optional request body.",
				},
				"scheme": map[string]interface{}{
					"type":        "string",
					"description": "http or https. Defaults to http.",
					"default":     "http",
				},
				"timeoutSeconds": map[string]interface{}{
					"type":        "integer",
					"description": "curl timeout in seconds. Defaults to 20.",
					"default":     20,
					"minimum":     1,
				},
			},
			"required": []string{"target", "service", "containerPort"},
		},
	}
}

func (s *server) callTool(params toolsCallParams) (toolResult, error) {
	switch params.Name {
	case "docker_compose_logs":
		var args composeLogsArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolResult{}, fmt.Errorf("invalid docker_compose_logs arguments: %w", err)
		}
		return s.composeLogs(args)
	case "compose_service_api_request":
		var args serviceRequestArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolResult{}, fmt.Errorf("invalid compose_service_api_request arguments: %w", err)
		}
		return s.serviceRequest(args)
	default:
		return toolResult{}, fmt.Errorf("unsupported tool: %s", params.Name)
	}
}

func (s *server) composeLogs(args composeLogsArgs) (toolResult, error) {
	projectDir, err := s.resolveTarget(args.Target)
	if err != nil {
		return toolResult{}, err
	}

	tail := args.Tail
	if tail <= 0 {
		tail = defaultComposeLogsTail
	} else if tail > maxComposeLogsTail {
		tail = maxComposeLogsTail
	}

	commandArgs, err := s.composeBaseArgs(projectDir, args.ComposeFile)
	if err != nil {
		return toolResult{}, err
	}
	commandArgs = append(commandArgs, "logs", "--no-color", "--tail", strconv.Itoa(tail))
	if args.Since != "" {
		commandArgs = append(commandArgs, "--since", args.Since)
	}
	if args.Timestamps {
		commandArgs = append(commandArgs, "--timestamps")
	}
	commandArgs = append(commandArgs, args.Services...)

	output, err := s.runRemoteCommand(projectDir, commandArgs)
	if err != nil {
		return toolResult{}, err
	}
	output = trimComposeLogNoise(output)

	return toolResult{Content: []toolTextContent{{Type: "text", Text: output}}}, nil
}

func (s *server) serviceRequest(args serviceRequestArgs) (toolResult, error) {
	projectDir, err := s.resolveTarget(args.Target)
	if err != nil {
		return toolResult{}, err
	}
	service := strings.TrimSpace(args.Service)
	if service == "" {
		return toolResult{}, errors.New("service is required")
	}
	if args.ContainerPort <= 0 {
		return toolResult{}, errors.New("containerPort must be greater than zero")
	}

	pathValue := strings.TrimSpace(args.Path)
	if pathValue == "" {
		pathValue = "/"
	}
	if !strings.HasPrefix(pathValue, "/") {
		return toolResult{}, errors.New("path must start with /")
	}

	scheme := strings.ToLower(strings.TrimSpace(args.Scheme))
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return toolResult{}, errors.New("scheme must be http or https")
	}

	method := strings.ToUpper(strings.TrimSpace(args.Method))
	if method == "" {
		method = "GET"
	}

	timeout := args.TimeoutSeconds
	if timeout <= 0 {
		timeout = 20
	}

	composeArgs, err := s.composeBaseArgs(projectDir, "")
	if err != nil {
		return toolResult{}, err
	}
	portOutput, err := s.runRemoteCommand(projectDir, append(composeArgs, "port", service, strconv.Itoa(args.ContainerPort)))
	if err != nil {
		return toolResult{}, fmt.Errorf("failed to resolve compose service port: %w", err)
	}

	hostPort, err := parsePublishedPort(portOutput)
	if err != nil {
		return toolResult{}, err
	}

	url := fmt.Sprintf("%s://127.0.0.1:%s%s", scheme, hostPort, pathValue)
	curlArgs := []string{
		"curl",
		"--silent",
		"--show-error",
		"--request", method,
		"--max-time", strconv.Itoa(timeout),
		"--write-out", "\n\nHTTP_STATUS:%{http_code}\n",
		url,
	}
	for key, value := range args.Headers {
		curlArgs = append(curlArgs, "--header", fmt.Sprintf("%s: %s", key, value))
	}
	if args.Body != "" {
		curlArgs = append(curlArgs, "--data", args.Body)
	}

	output, err := s.runRemoteCommand("", curlArgs)
	if err != nil {
		return toolResult{}, err
	}

	return toolResult{Content: []toolTextContent{{Type: "text", Text: output}}}, nil
}

func parsePublishedPort(output string) (string, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.LastIndex(line, ":")
		if idx == -1 || idx == len(line)-1 {
			return "", fmt.Errorf("unexpected docker compose port output: %s", line)
		}
		return line[idx+1:], nil
	}
	return "", errors.New("docker compose port returned no published port")
}

func (s *server) resolveTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", errors.New("target is required")
	}
	projectDir := strings.TrimSpace(s.config.allowedProjects[target])
	if projectDir == "" {
		return "", errors.New("target is not allowed")
	}
	return projectDir, nil
}

func (s *server) composeBaseArgs(projectDir string, composeFile string) ([]string, error) {
	commandArgs := []string{"docker", "compose"}
	if composeFile != "" {
		if filepath.IsAbs(composeFile) {
			return nil, errors.New("absolute composeFile paths are not allowed")
		}
		cleanComposeFile := filepath.Clean(composeFile)
		if cleanComposeFile == ".." || strings.HasPrefix(cleanComposeFile, "../") {
			return nil, errors.New("composeFile must stay within the allowed project directory")
		}
		composeFile = filepath.Join(projectDir, cleanComposeFile)
		commandArgs = append(commandArgs, "-f", composeFile)
	}
	return commandArgs, nil
}

func (s *server) runRemoteCommand(workDir string, commandArgs []string) (string, error) {
	if len(commandArgs) == 0 {
		return "", errors.New("remote command is empty")
	}

	var script strings.Builder
	script.WriteString("set -eu\n")
	if workDir != "" {
		script.WriteString("cd ")
		script.WriteString(shellPathExpr(workDir))
		script.WriteByte('\n')
	}
	script.WriteString("exec")
	for _, arg := range commandArgs {
		script.WriteByte(' ')
		script.WriteString(shellEscape(arg))
	}

	ctx, cancel := context.WithTimeout(context.Background(), sshCommandTimeoutSeconds*time.Second)
	defer cancel()

	sshArgs := []string{"-T"}
	if s.config.port != "" {
		sshArgs = append(sshArgs, "-p", s.config.port)
	}
	if s.config.identityFile != "" {
		sshArgs = append(sshArgs, "-i", s.config.identityFile)
	}
	if s.config.certificateFile != "" {
		sshArgs = append(sshArgs, "-o", "CertificateFile="+s.config.certificateFile)
	}
	if s.config.knownHostsFile != "" {
		sshArgs = append(sshArgs, "-o", "UserKnownHostsFile="+s.config.knownHostsFile)
	}
	sshArgs = append(
		sshArgs,
		s.config.target,
		"env",
		"-u", "BASH_ENV",
		"-u", "ENV",
		"/bin/sh",
		"-c",
		script.String(),
	)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	stdout := newLimitedBuffer(maxRemoteCommandOutput)
	stderr := newLimitedBuffer(maxRemoteCommandOutput)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("ssh command timed out after %d seconds", sshCommandTimeoutSeconds)
		}
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = err.Error()
		}
		return "", fmt.Errorf("remote command failed: %s", errText)
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		if stdout.Truncated() {
			return stdout.String(), nil
		}
		return "(no output)", nil
	}
	return text, nil
}

type limitedBuffer struct {
	limit     int
	buf       []byte
	truncated bool
}

func newLimitedBuffer(limit int) limitedBuffer {
	return limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	remaining := b.limit - len(b.buf)
	if remaining > 0 {
		if len(p) <= remaining {
			b.buf = append(b.buf, p...)
		} else {
			b.buf = append(b.buf, p[:remaining]...)
			b.truncated = true
		}
	} else {
		b.truncated = true
	}

	if remaining > 0 && len(p) > remaining {
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	text := string(b.buf)
	if b.truncated {
		return strings.TrimSpace(text) + "\n\n[output truncated]"
	}
	return text
}

func (b *limitedBuffer) Truncated() bool {
	return b.truncated
}

func shellEscape(v string) string {
	if v == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(v, "'", `'"'"'`) + "'"
}

func shellPathExpr(v string) string {
	switch {
	case v == "~":
		return "$HOME"
	case strings.HasPrefix(v, "~/"):
		return "$HOME/" + shellEscape(strings.TrimPrefix(v, "~/"))
	default:
		return shellEscape(v)
	}
}

func trimComposeLogNoise(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, " | ") {
			return strings.Join(lines[i:], "\n")
		}
	}
	return output
}

func errorResponse(id interface{}, code int, message string, data interface{}) *responseEnvelope {
	return &responseEnvelope{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
}

func decodeID(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}

	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		name, value, found := strings.Cut(line, ":")
		if !found {
			return nil, fmt.Errorf("invalid header line: %s", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}

	if contentLength <= 0 {
		return nil, errors.New("missing Content-Length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeFrame(w *bufio.Writer, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	return w.Flush()
}

func writeFatalProtocolError(w *bufio.Writer, err error) {
	_ = writeFrame(w, responseEnvelope{
		JSONRPC: "2.0",
		Error:   &rpcError{Code: -32603, Message: "protocol error", Data: err.Error()},
	})
}
