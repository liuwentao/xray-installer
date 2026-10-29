package config

import (
	"bytes"
	"image/png"
	"net/url"
	"strings"
	"testing"

	"github.com/skip2/go-qrcode"
)

func TestRenderShadowrocket(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"xx.example.com", "2001:db8::1"} {
		t.Run(host, func(t *testing.T) {
			params := sampleParameters()
			params.Domain = host
			params.Port = 8443
			params.NodeName = "测试节点 #1 & + / ?"
			uri, data, terminal, err := RenderShadowrocket(params)
			if err != nil {
				t.Fatal(err)
			}
			link, err := url.Parse(uri)
			if err != nil {
				t.Fatal(err)
			}
			if link.Scheme != "vless" || link.Hostname() != host || link.Port() != "8443" || link.User.Username() != params.UUID || link.Fragment != params.NodeName {
				t.Fatalf("unexpected share link: %s", uri)
			}
			for key, want := range map[string]string{
				"encryption": "none", "type": "tcp", "security": "reality",
				"flow": "xtls-rprx-vision", "sni": params.DestHost,
				"fp": "chrome", "pbk": params.PublicKey, "sid": params.ShortID, "spx": "/",
			} {
				if got := link.Query().Get(key); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
			if strings.Contains(uri, params.PrivateKey) || strings.ContainsAny(uri, "\r\n ") {
				t.Fatal("share link contains private key or unescaped whitespace")
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("invalid PNG: %v", err)
			}
			qr, err := qrcode.New(uri, qrcode.Medium)
			if err != nil {
				t.Fatal(err)
			}
			bitmap := qr.Bitmap()
			if img.Bounds().Dx() != len(bitmap)*8 || img.Bounds().Dy() != len(bitmap)*8 {
				t.Fatal("unexpected QR dimensions")
			}
			// Verify the saved image encodes exactly the exported URI, with a quiet zone.
			for y, row := range bitmap {
				for x, dark := range row {
					r, _, _, _ := img.At(x*8+4, y*8+4).RGBA()
					if (r == 0) != dark {
						t.Fatalf("QR module mismatch at %d,%d", x, y)
					}
				}
			}
			if !strings.Contains(terminal, "\x1b[30;47m") || !strings.HasSuffix(terminal, "\x1b[0m\n") {
				t.Fatal("terminal QR missing explicit colors or reset")
			}
		})
	}
}

func TestRenderShadowrocketTooLarge(t *testing.T) {
	params := sampleParameters()
	params.NodeName = strings.Repeat("x", 10000)
	if _, _, _, err := RenderShadowrocket(params); err == nil {
		t.Fatal("expected error for oversized QR payload")
	}
}
