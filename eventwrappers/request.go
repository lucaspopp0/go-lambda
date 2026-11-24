package eventwrappers

// HttpRequest is a common view over the HTTP request events delivered to Lambda.
type HttpRequest interface {
	// Method is the HTTP method, e.g. "GET".
	Method() string
	// Path is the raw request path, without the query string.
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
	// Unwrap returns the original event value (e.g. events.APIGatewayV2HTTPRequest).
	Unwrap() any
}

// UnwrapRequest returns the original event of type T behind req, and whether
// req actually wraps a T.
func UnwrapRequest[T any](req HttpRequest) (T, bool) {
	t, ok := req.Unwrap().(T)
	return t, ok
}
