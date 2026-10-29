package installer

import (
	"bytes"
	"strings"
	"testing"
)

func TestShadowrocketSummary(t *testing.T) {
	t.Parallel()
	result := &Result{
		ShadowrocketConfigPath: shadowrocketConfigPath,
		ShadowrocketQRPath:     shadowrocketQRPath,
	}
	var output bytes.Buffer
	inst := New(&output, &output)
	inst.printSummary(result)
	for _, want := range []string{"sudo cat " + shadowrocketConfigPath, shadowrocketQRPath, "小火箭"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("summary missing %q", want)
		}
	}
	output.Reset()
	inst.printDryRunSummary(result)
	for _, want := range []string{
		"would write shadowrocket config to: " + shadowrocketConfigPath,
		"would write shadowrocket QR code to: " + shadowrocketQRPath,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("dry-run summary missing %q", want)
		}
	}
	if strings.Contains(output.String(), "\x1b[") || strings.Contains(output.String(), "vless://") {
		t.Fatal("dry-run must not display an importable node")
	}
}
