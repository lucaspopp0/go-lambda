package eventwrappers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// HttpRequestEvent is the set of Lambda HTTP request events that can be wrapped.
type HttpRequestEvent interface {
	events.APIGatewayV2HTTPRequest | events.LambdaFunctionURLRequest
}

// HttpResponseEvent is the set of Lambda HTTP response events that can be wrapped.
type HttpResponseEvent interface {
	events.APIGatewayV2HTTPResponse | events.LambdaFunctionURLResponse
}

// WrapRequest wraps the given Lambda request event in the matching HttpRequest.
func WrapRequest[T HttpRequestEvent](event T) HttpRequest {
	switch e := any(event).(type) {
	case events.APIGatewayV2HTTPRequest:
		return WrapAPIGatewayV2Request(e)
	case events.LambdaFunctionURLRequest:
		return WrapFunctionURLRequest(e)
	}
	panic("eventwrappers: unreachable: unsupported request event type")
}

// WrapResponse wraps the given Lambda response event in the matching
// HttpResponse. The wrapper mutates the event; pass nil for a new, empty 200
// response.
func WrapResponse[T HttpResponseEvent](event *T) HttpResponse {
	switch e := any(event).(type) {
	case *events.APIGatewayV2HTTPResponse:
		return NewAPIGatewayV2Response(e)
	case *events.LambdaFunctionURLResponse:
		return NewFunctionURLResponse(e)
	}
	panic("eventwrappers: unreachable: unsupported response event type")
}

// NewResponseFor creates a new, empty 200 response of the event type that
// matches req (e.g. an API Gateway v2 request gets an API Gateway v2 response).
// It returns nil if req does not wrap a supported event type.
func NewResponseFor(req HttpRequest) HttpResponse {
	switch req.Unwrap().(type) {
	case events.APIGatewayV2HTTPRequest:
		return NewAPIGatewayV2Response(nil)
	case events.LambdaFunctionURLRequest:
		return NewFunctionURLResponse(nil)
	}
	return nil
}

// ToHttpRequest wraps the given Lambda request event and converts it into an
// *http.Request, by calling HttpRequest.ToHttpRequest on the wrapper.
func ToHttpRequest[T HttpRequestEvent](event T) (*http.Request, error) {
	return WrapRequest(event).ToHttpRequest()
}

// FromHttpResponse creates a new, empty Lambda response event of type T and
// fills it from the given HTTP response parts, by calling
// HttpResponse.FromHttpResponse on the wrapper.
func FromHttpResponse[T HttpResponseEvent](code int, header http.Header, body []byte) T {
	var event T
	WrapResponse(&event).FromHttpResponse(code, header, body)
	return event
}

// HttpRequest is a common view over the HTTP request events delivered to Lambda.
type HttpRequest interface {
	// Method is the HTTP method, e.g. "GET".
	Method() string
	// Path is the raw request path, without the query string. For API Gateway v2
	// events the stage prefix is already stripped.
	Path() string
	// RawQuery is the raw (still-encoded) query string, without a leading "?".
	RawQuery() string
	// Headers returns a copy of the request headers, keyed as delivered by Lambda.
	Headers() map[string]string
	// Header returns the value of the named header (case-insensitive), or "".
	Header(name string) string
	// Cookies returns the request cookies as "name=value" strings.
	Cookies() []string
	// Body returns the request body, base64-decoding it if needed.
	Body() ([]byte, error)
	// ToHttpRequest converts the event into an *http.Request.
	ToHttpRequest() (*http.Request, error)
	// Unwrap returns the original event value (e.g. events.APIGatewayV2HTTPRequest).
	Unwrap() any
}

// UnwrapRequest returns the original event of type T behind req, and whether
// req actually wraps a T.
func UnwrapRequest[T any](req HttpRequest) (T, bool) {
	t, ok := req.Unwrap().(T)
	return t, ok
}

// HttpResponse is a common, mutable view over the HTTP response events returned
// from Lambda handlers.
type HttpResponse interface {
	// StatusCode returns the HTTP status code.
	StatusCode() int
	// SetStatusCode sets the HTTP status code.
	SetStatusCode(code int)
	// Headers returns a copy of the response headers (excluding cookies).
	Headers() map[string]string
	// Header returns the value of the named header (case-insensitive), or "".
	Header(name string) string
	// SetHeader replaces the named header. Set-Cookie values go to the cookies.
	SetHeader(name, value string)
	// AddHeader appends a value to the named header. Where the event only holds
	// one value per header, values are joined with ", ". Set-Cookie values are
	// appended to the cookies.
	AddHeader(name, value string)
	// Cookies returns the Set-Cookie values.
	Cookies() []string
	// Body returns the response body, base64-decoding it if needed.
	Body() ([]byte, error)
	// SetBody sets a text body.
	SetBody(body string)
	// SetBinaryBody sets a binary body, base64-encoding it in the event.
	SetBinaryBody(body []byte)
	// FromHttpResponse replaces the status, headers, cookies and body with the
	// given HTTP response parts. Binary content types are base64-encoded.
	// Multi-valued headers are represented as the event allows: API Gateway v2
	// keeps the last value in Headers and all values in MultiValueHeaders, while
	// Function URLs join them with ", ". Set-Cookie values go to the cookies
	// on Function URLs.
	FromHttpResponse(code int, header http.Header, body []byte)
	// Unwrap returns the original event value (e.g. events.APIGatewayV2HTTPResponse).
	Unwrap() any
}

// UnwrapResponse returns the original event of type T behind resp, and whether
// resp actually wraps a T.
func UnwrapResponse[T any](resp HttpResponse) (T, bool) {
	t, ok := resp.Unwrap().(T)
	return t, ok
}

// buildHttpRequest converts a wrapped Lambda HTTP event into an *http.Request.
func buildHttpRequest(req HttpRequest) (*http.Request, error) {
	body, err := req.Body()
	if err != nil {
		return nil, err
	}

	requestURL := req.Path()
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

// setHttpBody stores body as text, or base64-encoded for binary content types.
func setHttpBody(resp HttpResponse, contentType string, body []byte) {
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
