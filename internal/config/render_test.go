package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func sampleParameters() Parameters {
	return Parameters{
		Domain:     "xx.example.com",
		NodeName:   `测试节点 "A"`,
		UUID:       "12345678-1234-1234-1234-1234567890ab",
		PrivateKey: "private-key",
		PublicKey:  "public-key",
		ShortID:    "0123456789abcdef",
		DestHost:   "www.apple.com",
		Port:       DefaultPort,
	}
}

func TestRenderXray(t *testing.T) {
	data, err := RenderXray(sampleParameters())
	if err != nil {
		t.Fatalf("RenderXray returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("rendered xray config is not valid json: %v", err)
	}

	text := string(data)
	for _, want := range []string{
		`"id": "12345678-1234-1234-1234-1234567890ab"`,
		`"privateKey": "private-key"`,
		`"shortIds": [`,
		`"www.apple.com:443"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered xray config missing %q", want)
		}
	}
}

func TestRenderXrayClient(t *testing.T) {
	params := sampleParameters()
	params.Port = 8443
	data, err := RenderXrayClient(params)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Inbounds []struct {
			Listen   string
			Port     int
			Protocol string
		}
		Outbounds []struct {
			Protocol string
			Settings struct {
				Vnext []struct {
					Address string
					Port    int
					Users   []struct{ ID, Encryption, Flow string }
				}
			}
			StreamSettings struct {
				Security        string
				RealitySettings struct{ ServerName, Password, ShortID, Fingerprint string }
			}
		}
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Inbounds) != 2 || len(parsed.Outbounds) != 1 {
		t.Fatalf("unexpected client structure: %s", data)
	}
	for n, inbound := range parsed.Inbounds {
		if inbound.Listen != "127.0.0.1" || inbound.Port != []int{1080, 8080}[n] || inbound.Protocol != []string{"socks", "http"}[n] {
			t.Fatalf("unexpected local proxy: %+v", inbound)
		}
	}
	outbound := parsed.Outbounds[0]
	if outbound.Protocol != "vless" || len(outbound.Settings.Vnext) != 1 {
		t.Fatalf("unexpected outbound: %+v", outbound)
	}
	node := outbound.Settings.Vnext[0]
	if node.Address != params.Domain || node.Port != params.Port || len(node.Users) != 1 {
		t.Fatalf("unexpected node: %+v", node)
	}
	user := node.Users[0]
	reality := outbound.StreamSettings.RealitySettings
	if user.ID != params.UUID || user.Encryption != "none" || user.Flow != "xtls-rprx-vision" || outbound.StreamSettings.Security != "reality" || reality.ServerName != params.DestHost || reality.Password != params.PublicKey || reality.ShortID != params.ShortID || reality.Fingerprint != "chrome" {
		t.Fatalf("client credentials do not match server parameters: %s", data)
	}
	if strings.Contains(string(data), params.PrivateKey) || strings.Contains(string(data), "privateKey") {
		t.Fatal("client config contains server private key")
	}
}

func TestRenderFlClash(t *testing.T) {
	data, err := RenderFlClash(sampleParameters())
	if err != nil {
		t.Fatalf("RenderFlClash returned error: %v", err)
	}

	want, err := os.ReadFile("testdata/proxy.yaml.golden")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	if string(data) != string(want) {
		t.Fatalf("rendered flclash config mismatch.\n--- got ---\n%s\n--- want ---\n%s", string(data), string(want))
	}
}

func TestRenderMihomo(t *testing.T) {
	t.Parallel()
	params := sampleParameters()
	params.NodeName = "测试节点 \"A\": #1\n第二行"
	data, err := RenderMihomo(params)
	if err != nil {
		t.Fatalf("RenderMihomo returned error: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"mixed-port: 7890", "allow-lan: false", "mode: rule",
		"type: vless", "flow: xtls-rprx-vision", "packet-encoding: xudp",
		"tls: true", "client-fingerprint: chrome",
		"support-x25519mlkem768: true", "type: select", "MATCH,PROXY",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("Mihomo config missing %q", want)
		}
	}
	// JSON-quoted scalars are also YAML scalars; check that credentials and
	// names survive quoting, including names referenced by the proxy group.
	for _, tt := range []struct {
		prefix string
		want   string
	}{
		{"  - name: ", params.NodeName},
		{"    server: ", params.Domain},
		{"    uuid: ", params.UUID},
		{"    servername: ", params.DestHost},
		{"      public-key: ", params.PublicKey},
		{"      short-id: ", params.ShortID},
		{"      - ", params.NodeName},
	} {
		found := false
		for _, line := range strings.Split(text, "\n") {
			if !strings.HasPrefix(line, tt.prefix) {
				continue
			}
			var got string
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, tt.prefix)), &got); err == nil && got == tt.want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing correctly quoted %q value %q", tt.prefix, tt.want)
		}
	}
	if strings.Contains(text, params.PrivateKey) || strings.Contains(text, "private-key:") {
		t.Fatal("client config must not contain the server private key")
	}
	params.Port = 8443
	data, err = RenderMihomo(params)
	if err != nil || !strings.Contains(string(data), "    port: 8443\n") {
		t.Fatalf("custom port not rendered: %v", err)
	}
}
