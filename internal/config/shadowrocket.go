package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
)

// RenderShadowrocket renders a VLESS share URI and matching local QR codes.
// Only client parameters are included; the server private key is never exported.
func RenderShadowrocket(params Parameters) (uri string, png []byte, terminal string, err error) {
	query := url.Values{
		"encryption": {"none"},
		"type":       {"tcp"},
		"security":   {"reality"},
		"flow":       {"xtls-rprx-vision"},
		"sni":        {params.DestHost},
		"fp":         {"chrome"},
		"pbk":        {params.PublicKey},
		"sid":        {params.ShortID},
		"spx":        {"/"},
	}
	link := url.URL{
		Scheme:   "vless",
		User:     url.User(params.UUID),
		Host:     net.JoinHostPort(params.Domain, strconv.Itoa(params.Port)),
		RawQuery: query.Encode(),
		Fragment: params.NodeName,
	}
	uri = link.String()
	qr, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return "", nil, "", fmt.Errorf("encode shadowrocket QR code: %w", err)
	}
	// An integer scale keeps modules sharp, including the four-module quiet zone.
	png, err = qr.PNG(-8)
	if err != nil {
		return "", nil, "", fmt.Errorf("render shadowrocket QR code: %w", err)
	}
	// Explicit colors keep the QR readable on both light and dark terminals.
	var out strings.Builder
	bitmap := qr.Bitmap()
	for y := 0; y < len(bitmap); y += 2 {
		out.WriteString("\x1b[30;47m")
		for x, top := range bitmap[y] {
			bottom := y+1 < len(bitmap) && bitmap[y+1][x]
			switch {
			case top && bottom:
				out.WriteRune('█')
			case top:
				out.WriteRune('▀')
			case bottom:
				out.WriteRune('▄')
			default:
				out.WriteByte(' ')
			}
		}
		out.WriteString("\x1b[0m\n")
	}
	return uri, png, out.String(), nil
}
