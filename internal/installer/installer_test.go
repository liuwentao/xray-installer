package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xray-installer/internal/config"
)

func TestSupportsDistro(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		like string
		want bool
	}{
		{name: "ubuntu", id: "ubuntu", want: true},
		{name: "debian", id: "debian", want: true},
		{name: "archlinux id", id: "archlinux", want: true},
		{name: "arch id", id: "arch", want: true},
		{name: "arch like", id: "manjaro", like: "archlinux", want: true},
		{name: "unsupported", id: "alpine", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := supportsDistro(tt.id, tt.like); got != tt.want {
				t.Fatalf("supportsDistro(%q, %q) = %v, want %v", tt.id, tt.like, got, tt.want)
			}
		})
	}
}

func TestPrintSummaryIncludesFlClashCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	inst := New(&stdout, &stdout)
	inst.printSummary(&Result{
		Domain:           "xx.example.com",
		NodeName:         DefaultNodeName,
		PublicIP:         "1.2.3.4",
		Port:             443,
		UUID:             "uuid",
		PublicKey:        "pub",
		ShortID:          "short",
		XrayConfigPath:   xrayConfigPath,
		ProxyConfigPath:  proxyConfigPath,
		MihomoConfigPath: mihomoConfigPath,
	})

	if !strings.Contains(stdout.String(), "sudo cat "+proxyConfigPath) {
		t.Fatalf("summary missing flclash fetch command: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "sudo cat "+mihomoConfigPath) {
		t.Fatalf("summary missing mihomo fetch command: %q", stdout.String())
	}
}

func TestPrintDryRunSummaryIncludesFlClashCommandHint(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	inst := New(&stdout, &stdout)
	inst.printDryRunSummary(&Result{
		Domain:           "xx.example.com",
		NodeName:         DefaultNodeName,
		PublicIP:         "1.2.3.4",
		XrayConfigPath:   xrayConfigPath,
		ProxyConfigPath:  proxyConfigPath,
		MihomoConfigPath: mihomoConfigPath,
	})

	if !strings.Contains(stdout.String(), "sudo cat "+proxyConfigPath) {
		t.Fatalf("dry-run summary missing flclash fetch command hint: %q", stdout.String())
	}
	for _, want := range []string{"would write mihomo config to: " + mihomoConfigPath, "sudo cat " + mihomoConfigPath} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("dry-run summary missing %q: %q", want, stdout.String())
		}
	}
}

func TestWriteFileAtomicUsesRequestedMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "config.json")
	if err := writeFileAtomic(target, []byte(`{"ok":true}`), xrayConfigMode); err != nil {
		t.Fatalf("writeFileAtomic returned error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != xrayConfigMode {
		t.Fatalf("file mode = %o, want %o", got, xrayConfigMode)
	}
}

func TestNormalizeDestHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "empty uses default",
			input: "",
			want:  config.DefaultDestHost,
		},
		{
			name:  "space uses default",
			input: "   ",
			want:  config.DefaultDestHost,
		},
		{
			name:  "custom domain normalized",
			input: "WWW.Apple.COM.",
			want:  "www.apple.com",
		},
		{
			name:    "invalid domain",
			input:   "not a domain",
			wantErr: "invalid domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDestHost(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("normalizeDestHost error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDestHost returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeDestHost = %q, want %q", got, tt.want)
			}
		})
	}
}
