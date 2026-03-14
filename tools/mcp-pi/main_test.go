package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestShellEscape(t *testing.T) {
	t.Parallel()

	got := shellEscape(`a'b c`)
	want := `'a'"'"'b c'`
	if got != want {
		t.Fatalf("shellEscape() = %q, want %q", got, want)
	}
}

func TestParsePublishedPort(t *testing.T) {
	t.Parallel()

	got, err := parsePublishedPort("0.0.0.0:6001\n")
	if err != nil {
		t.Fatalf("parsePublishedPort() error = %v", err)
	}
	if got != "6001" {
		t.Fatalf("parsePublishedPort() = %q, want %q", got, "6001")
	}
}

func TestWriteAndReadFrame(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "ping",
	}

	if err := writeFrame(writer, payload); err != nil {
		t.Fatalf("writeFrame() error = %v", err)
	}

	reader := bufio.NewReader(strings.NewReader(buf.String()))
	body, err := readFrame(reader)
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}

	if !strings.Contains(string(body), `"method":"ping"`) {
		t.Fatalf("unexpected body %q", string(body))
	}
}

func TestResolveTarget(t *testing.T) {
	t.Parallel()

	const (
		projectOne = "~/project-one"
		projectTwo = "~/project-two"
	)

	s := &server{config: sshConfig{allowedProjects: map[string]string{
		"dir1": projectOne,
		"dir2": projectTwo,
	}}}

	got, err := s.resolveTarget("dir1")
	if err != nil {
		t.Fatalf("resolveTarget() unexpected error = %v", err)
	}
	if got != projectOne {
		t.Fatalf("resolveTarget() = %q, want %q", got, projectOne)
	}

	if _, err := s.resolveTarget("other"); err == nil {
		t.Fatal("resolveTarget() expected error for invalid target")
	}
}

func TestComposeBaseArgsRejectsEscapingPaths(t *testing.T) {
	t.Parallel()

	s := &server{}
	const projectDir = "~/project-one"

	if _, err := s.composeBaseArgs(projectDir, "/tmp/docker-compose.yml"); err == nil {
		t.Fatal("composeBaseArgs() expected error for absolute composeFile")
	}
	if _, err := s.composeBaseArgs(projectDir, "../docker-compose.yml"); err == nil {
		t.Fatal("composeBaseArgs() expected error for parent-relative composeFile")
	}
}

func TestTrimComposeLogNoise(t *testing.T) {
	t.Parallel()

	input := "HOME='/home/test-user'\nPATH='/usr/bin'\nscheduler-1  | log line"
	got := trimComposeLogNoise(input)
	if got != "scheduler-1  | log line" {
		t.Fatalf("trimComposeLogNoise() = %q", got)
	}
}
