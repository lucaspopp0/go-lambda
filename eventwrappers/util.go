package eventwrappers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidBody is returned (wrapped) when an event body cannot be decoded,
// e.g. it is flagged as base64 but is not valid base64. It indicates a bad
// request rather than a server fault.
var ErrInvalidBody = errors.New("invalid request body")

func decodeBody(body string, isBase64 bool) ([]byte, error) {
	if !isBase64 {
		return []byte(body), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBody, err)
	}
	return decoded, nil
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
