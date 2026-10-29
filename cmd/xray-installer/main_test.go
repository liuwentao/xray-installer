package main

import (
	"bytes"
	"strings"
	"testing"

	"xray-installer/internal/config"
)

func TestParseCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    cliOptions
		wantErr string
	}{
		{
			name: "install default",
			args: nil,
			want: cliOptions{command: commandInstall},
		},
		{
			name: "install dry-run",
			args: []string{"--dry-run"},
			want: cliOptions{command: commandInstall, dryRun: true},
		},
		{
			name: "uninstall dry-run",
			args: []string{"uninstall", "--dry-run"},
			want: cliOptions{command: commandUninstall, dryRun: true},
		},
		{
			name: "help",
			args: []string{"--help"},
			want: cliOptions{command: commandInstall, help: true},
		},
		{
			name: "update latest",
			args: []string{"update"},
			want: cliOptions{command: commandUpdate},
		},
		{
			name: "update specific version dry-run",
			args: []string{"update", "--xray-version", "v26.3.27", "--dry-run"},
			want: cliOptions{command: commandUpdate, xrayVersion: "v26.3.27", dryRun: true},
		},
		{
			name: "update help",
			args: []string{"update", "--help"},
			want: cliOptions{command: commandUpdate, help: true},
		},
		{
			name: "update installer version",
			args: []string{"update", "--version"},
			want: cliOptions{command: commandUpdate, version: true},
		},
		{
			name:    "missing xray version",
			args:    []string{"update", "--xray-version"},
			wantErr: "needs an argument",
		},
		{
			name:    "xray version only for update",
			args:    []string{"--xray-version", "v26.3.27"},
			wantErr: "flag provided but not defined",
		},
		{
			name: "version",
			args: []string{"--version"},
			want: cliOptions{command: commandInstall, version: true},
		},
		{
			name:    "unexpected arg",
			args:    []string{"example.com"},
			wantErr: "unexpected arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCLI(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCLI error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCLI returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseCLI = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPromptInstallRequest(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("xx.example.com\nwww.apple.com\n")
	var output bytes.Buffer

	req, err := promptInstallRequest(input, &output)
	if err != nil {
		t.Fatalf("promptInstallRequest returned error: %v", err)
	}

	if req.Domain != "xx.example.com" {
		t.Fatalf("Domain = %q, want xx.example.com", req.Domain)
	}
	if req.DestHost != "www.apple.com" {
		t.Fatalf("DestHost = %q, want www.apple.com", req.DestHost)
	}
	if !strings.Contains(output.String(), "请输入 VPS 绑定的域名") {
		t.Fatalf("prompt output missing expected labels: %q", output.String())
	}
	if !strings.Contains(output.String(), "请输入伪装目标域名") {
		t.Fatalf("prompt output missing dest host label: %q", output.String())
	}
	if strings.Contains(output.String(), "请输入 FlClash 节点名称") {
		t.Fatalf("prompt output should not ask for node name: %q", output.String())
	}
}

func TestPromptInstallRequestUsesDefaultDestHost(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("xx.example.com\n\n")
	var output bytes.Buffer

	req, err := promptInstallRequest(input, &output)
	if err != nil {
		t.Fatalf("promptInstallRequest returned error: %v", err)
	}

	if req.DestHost != config.DefaultDestHost {
		t.Fatalf("DestHost = %q, want %q", req.DestHost, config.DefaultDestHost)
	}
	if !strings.Contains(output.String(), "直接回车默认 "+config.DefaultDestHost) {
		t.Fatalf("prompt output missing default dest host hint: %q", output.String())
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	oldVersion, oldCommit, oldBuildDate := version, commit, buildDate
	version, commit, buildDate = "v1.2.3", "abc123", "2026-04-09T00:00:00Z"
	defer func() {
		version, commit, buildDate = oldVersion, oldCommit, oldBuildDate
	}()

	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run returned code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"xray-installer v1.2.3", "commit: abc123", "build date: 2026-04-09T00:00:00Z"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q in %q", want, stdout.String())
		}
	}
}

func TestRunUpdateInvalidVersionDoesNotPrompt(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"update", "--xray-version", "--force"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "invalid Xray version") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "请输入") {
		t.Fatalf("update must not prompt for installation: %q", stdout.String())
	}
}
