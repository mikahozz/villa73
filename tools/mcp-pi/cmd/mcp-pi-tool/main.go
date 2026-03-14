package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type requestEnvelope struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type responseEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type toolResponse struct {
	Content []toolTextContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}

type toolTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "logs":
		if err := runLogs(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "api":
		if err := runAPI(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}

func runLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var services stringListFlag
	defaultTarget := strings.TrimSpace(os.Getenv("MCP_PI_DEFAULT_TARGET"))
	target := fs.String("target", defaultTarget, "Configured target name: dir1 or dir2")
	composeFile := fs.String("compose-file", "", "Optional compose file path")
	tail := fs.Int("tail", 200, "Log line count")
	since := fs.String("since", "", "Docker log since value")
	timestamps := fs.Bool("timestamps", false, "Include timestamps")
	fs.Var(&services, "service", "Compose service name, repeatable")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *target == "" {
		return errors.New("logs requires --target or MCP_PI_DEFAULT_TARGET")
	}

	payload := map[string]any{
		"target":     *target,
		"services":   []string(services),
		"tail":       *tail,
		"since":      *since,
		"timestamps": *timestamps,
	}
	if *composeFile != "" {
		payload["composeFile"] = *composeFile
	}

	return callTool("docker_compose_logs", payload)
}

func runAPI(args []string) error {
	fs := flag.NewFlagSet("api", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var headers stringListFlag
	defaultTarget := strings.TrimSpace(os.Getenv("MCP_PI_DEFAULT_TARGET"))
	target := fs.String("target", defaultTarget, "Configured target name: dir1 or dir2")
	service := fs.String("service", "", "Compose service name")
	containerPort := fs.Int("container-port", 0, "Exposed container port")
	pathValue := fs.String("path", "/", "HTTP path")
	method := fs.String("method", "GET", "HTTP method")
	body := fs.String("body", "", "HTTP request body")
	scheme := fs.String("scheme", "http", "http or https")
	timeout := fs.Int("timeout", 20, "Timeout in seconds")
	fs.Var(&headers, "header", "Header in 'Name: Value' form, repeatable")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *target == "" {
		return errors.New("api requires --target or MCP_PI_DEFAULT_TARGET")
	}
	if *service == "" {
		return errors.New("api requires --service")
	}
	if *containerPort <= 0 {
		return errors.New("api requires --container-port")
	}

	headerMap := make(map[string]string, len(headers))
	for _, header := range headers {
		name, value, found := strings.Cut(header, ":")
		if !found {
			return fmt.Errorf("invalid header %q", header)
		}
		headerMap[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}

	payload := map[string]any{
		"target":         *target,
		"service":        *service,
		"containerPort":  *containerPort,
		"path":           *pathValue,
		"method":         *method,
		"body":           *body,
		"scheme":         *scheme,
		"timeoutSeconds": *timeout,
	}
	if len(headerMap) > 0 {
		payload["headers"] = headerMap
	}

	return callTool("compose_service_api_request", payload)
}

func callTool(name string, arguments map[string]any) error {
	serverCommand := strings.TrimSpace(os.Getenv("MCP_PI_SERVER_CMD"))
	if serverCommand == "" {
		return errors.New("MCP_PI_SERVER_CMD is not set")
	}

	cmd := exec.Command(serverCommand)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	reader := bufio.NewReader(stdoutPipe)
	if err := writeFrame(stdin, requestEnvelope{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  map[string]string{"protocolVersion": "2024-11-05"},
	}); err != nil {
		return err
	}
	if _, err := readFrame(reader); err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	if err := writeFrame(stdin, requestEnvelope{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}); err != nil {
		return err
	}

	if err := writeFrame(stdin, requestEnvelope{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}); err != nil {
		return err
	}
	_ = stdin.Close()

	frame, err := readFrame(reader)
	if err != nil {
		return fmt.Errorf("failed to read tool response: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = err.Error()
		}
		return fmt.Errorf("MCP server process failed: %s", errText)
	}

	var resp responseEnvelope
	if err := json.Unmarshal(frame, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var toolResp toolResponse
	if err := json.Unmarshal(resp.Result, &toolResp); err != nil {
		return err
	}
	if toolResp.IsError {
		return errors.New(joinToolText(toolResp.Content))
	}

	fmt.Println(joinToolText(toolResp.Content))
	return nil
}

func joinToolText(content []toolTextContent) string {
	parts := make([]string, 0, len(content))
	for _, item := range content {
		if item.Type == "text" {
			parts = append(parts, item.Text)
		}
	}
	return strings.Join(parts, "\n")
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

func writeFrame(w io.Writer, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  mcp-pi-tool logs --target dir1 --service scheduler --tail 120 --since 36h --timestamps")
	fmt.Fprintln(os.Stderr, "  mcp-pi-tool api --target dir2 --service scheduler --container-port 6002 --path /health")
}
