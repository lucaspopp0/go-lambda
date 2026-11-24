package eventwrappers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

// buildHTTPRequest converts a wrapped Lambda HTTP event into an *http.Request,
// using path in place of req.Path() so callers can adjust it.
func buildHTTPRequest(req HttpRequest, path string) (*http.Request, error) {
	body, err := req.Body()
	if err != nil {
		return nil, err
	}

	requestURL := path
	if rawQuery := req.RawQuery(); rawQuery != "" {
		requestURL = fmt.Sprintf("%s?%s", requestURL, rawQuery)
	}

	request, err := http.NewRequest(req.Method(), requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for name, value := range req.Headers() {
		request.Header.Set(name, value)
	}

	if cookies := req.Cookies(); len(cookies) > 0 {
		request.Header.Set("Cookie", strings.Join(cookies, "; "))
	}

	return request, nil
}

// setHTTPBody stores body as text, or base64-encoded for binary content types.
func setHTTPBody(resp HttpResponse, contentType string, body []byte) {
	if isBinaryContentType(contentType) {
		resp.SetBinaryBody(body)
	} else {
		resp.SetBody(string(body))
	}
}

func isBinaryContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") ||
		contentType == "application/octet-stream"
}
