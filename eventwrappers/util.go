package eventwrappers

import (
	"encoding/base64"
	"strings"
)

func decodeBody(body string, isBase64 bool) ([]byte, error) {
	if !isBase64 {
		return []byte(body), nil
	}
	return base64.StdEncoding.DecodeString(body)
}

func encodeBody(body []byte) string {
	return base64.StdEncoding.EncodeToString(body)
}

func copyHeaders(h map[string]string) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}

func lookupHeader(h map[string]string, name string) string {
	if v, ok := h[name]; ok {
		return v
	}
	for k, v := range h {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}

func isSetCookie(name string) bool {
	return strings.EqualFold(name, "Set-Cookie")
}

func deleteHeader[V any](h map[string]V, name string) {
	for k := range h {
		if strings.EqualFold(k, name) {
			delete(h, k)
		}
	}
}
