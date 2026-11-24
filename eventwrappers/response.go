package eventwrappers

import "net/http"

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
	// FromHTTPResponse replaces the status, headers, cookies and body with the
	// given HTTP response parts. Binary content types are base64-encoded.
	// Multi-valued headers are represented as the event allows: API Gateway v2
	// keeps the last value in Headers and all values in MultiValueHeaders, while
	// Function URLs join them with ", ". Set-Cookie values go to the cookies
	// on Function URLs.
	FromHTTPResponse(code int, header http.Header, body []byte)
	// Unwrap returns the original event value (e.g. events.APIGatewayV2HTTPResponse).
	Unwrap() any
}

// UnwrapResponse returns the original event of type T behind resp, and whether
// resp actually wraps a T.
func UnwrapResponse[T any](resp HttpResponse) (T, bool) {
	t, ok := resp.Unwrap().(T)
	return t, ok
}
