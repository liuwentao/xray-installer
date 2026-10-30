package installer

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeXrayVersion(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		input   string
		want    string
		invalid bool
	}{
		{input: "", want: ""},
		{input: "26.3.27", want: "v26.3.27"},
		{input: " v26.3.27 ", want: "v26.3.27"},
		{input: "v26.3.27-beta.1", want: "v26.3.27-beta.1"},
		{input: "latest", invalid: true},
		{input: "--force", invalid: true},
		{input: "v26.3", invalid: true},
		{input: "../../etc", invalid: true},
		{input: "v26.3.27; reboot", invalid: true},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeXrayVersion(tt.input)
			if (err != nil) != tt.invalid || got != tt.want {
				t.Fatalf("normalizeXrayVersion(%q) = %q, %v; want %q, invalid=%v", tt.input, got, err, tt.want, tt.invalid)
			}
		})
	}
}

func TestUpdateRejectsInvalidVersionBeforeSystemChanges(t *testing.T) {
	t.Parallel()
	inst := New(io.Discard, io.Discard)
	if err := inst.Update(context.Background(), UpdateRequest{Version: "--force"}); err == nil {
		t.Fatal("expected invalid version error")
	}
}

func TestRenderMissingXrayClient(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	client := filepath.Join(dir, "xray-client.json")
	record := filepath.Join(dir, "install-result.json")
	result := Result{Domain: "node.example.com", UUID: "existing-uuid", PublicKey: "existing-public-key", ShortID: "0123456789abcdef", DestHost: "www.bing.com", Port: 8443}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(record, data, privateMode); err != nil {
		t.Fatal(err)
	}
	got, err := renderMissingXrayClient(client, record)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Fatalf("invalid JSON: %s", got)
	}
	for _, want := range []string{result.Domain, result.UUID, result.PublicKey, result.ShortID, result.DestHost, "8443"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing existing parameter %q", want)
		}
	}
	if _, err := os.Stat(client); !os.IsNotExist(err) {
		t.Fatal("render must not write client config (dry-run)")
	}
	after, err := os.ReadFile(record)
	if err != nil || string(after) != string(data) {
		t.Fatal("install record was modified")
	}
	if err := os.WriteFile(client, []byte("custom client config"), privateMode); err != nil {
		t.Fatal(err)
	}
	got, err = renderMissingXrayClient(client, filepath.Join(dir, "missing-record.json"))
	if err != nil || got != nil {
		t.Fatalf("existing client must be preserved without requiring metadata: %s, %v", got, err)
	}
	after, err = os.ReadFile(client)
	if err != nil || string(after) != "custom client config" {
		t.Fatal("existing client changed")
	}
}

func TestRenderMissingXrayClientRejectsBadRecords(t *testing.T) {
	t.Parallel()
	for _, record := range []string{"", "{", "{}", `{"domain":"node.example.com","port":443}`} {
		t.Run(record, func(t *testing.T) {
			dir := t.TempDir()
			recordPath := filepath.Join(dir, "install-result.json")
			if record != "" {
				if err := os.WriteFile(recordPath, []byte(record), privateMode); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := renderMissingXrayClient(filepath.Join(dir, "client.json"), recordPath); err == nil {
				t.Fatal("expected invalid or missing record error")
			}
		})
	}
}
