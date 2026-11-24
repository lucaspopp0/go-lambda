package eventwrappers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

// stagePrefixer is implemented by request wrappers whose path carries a stage
// prefix (e.g. "/prod") that must be removed before routing.
type stagePrefixer interface {
	stage() string
}

// ToHTTPRequest converts a wrapped Lambda HTTP event into an *http.Request.
// For API Gateway v2 events, the stage prefix is stripped from the path.
func ToHTTPRequest(req HttpRequest) (*http.Request, error) {
	body, err := req.Body()
	if err != nil {
		return nil, err
	}

	requestURL := req.Path()
	if sp, ok := req.(stagePrefixer); ok && sp.stage() != "" {
		requestURL = strings.TrimPrefix(requestURL, "/"+sp.stage())
	}
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

// APIGatewayV2ResponseFromHTTP builds an API Gateway v2 response from the parts
// of an HTTP response. Headers carry the last value of each header, and
// MultiValueHeaders carries all of them.
func APIGatewayV2ResponseFromHTTP(code int, header http.Header, body []byte) HttpResponse {
	resp := NewAPIGatewayV2Response(nil).(*apiGatewayV2Response)
	resp.event.StatusCode = code
	resp.event.Headers = map[string]string{}
	resp.event.MultiValueHeaders = map[string][]string{}

	for name, values := range header {
		if len(values) > 0 {
			resp.event.Headers[name] = values[len(values)-1]
			resp.event.MultiValueHeaders[name] = values
		}
	}

	setHTTPBody(resp, resp.event.Headers["Content-Type"], body)
	return resp
}

// FunctionURLResponseFromHTTP builds a Lambda Function URL response from the
// parts of an HTTP response. Function URLs only support single-valued headers,
// so multiple values are joined with ", " and Set-Cookie values go to Cookies.
func FunctionURLResponseFromHTTP(code int, header http.Header, body []byte) HttpResponse {
	resp := NewFunctionURLResponse(nil).(*functionURLResponse)
	resp.event.StatusCode = code
	resp.event.Headers = map[string]string{}

	for name, values := range header {
		if len(values) == 0 {
			continue
		}
		if http.CanonicalHeaderKey(name) == "Set-Cookie" {
			resp.event.Cookies = append(resp.event.Cookies, values...)
			continue
		}
		resp.event.Headers[name] = strings.Join(values, ", ")
	}

	setHTTPBody(resp, resp.event.Headers["Content-Type"], body)
	return resp
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
