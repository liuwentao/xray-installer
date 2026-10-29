package installer

import (
	"context"
	"io"
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
