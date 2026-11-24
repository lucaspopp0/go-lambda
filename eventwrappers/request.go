package eventwrappers

import (
	"encoding/base64"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

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

// WrapAPIGatewayV2Request wraps an API Gateway v2 (HTTP API) request event.
func WrapAPIGatewayV2Request(event events.APIGatewayV2HTTPRequest) HttpRequest {
	return &apiGatewayV2Request{event: event}
}

// WrapFunctionURLRequest wraps a Lambda Function URL request event.
func WrapFunctionURLRequest(event events.LambdaFunctionURLRequest) HttpRequest {
	return &functionURLRequest{event: event}
}

type apiGatewayV2Request struct {
	event events.APIGatewayV2HTTPRequest
}

func (r *apiGatewayV2Request) Method() string             { return r.event.RequestContext.HTTP.Method }
func (r *apiGatewayV2Request) Path() string               { return r.event.RawPath }
func (r *apiGatewayV2Request) RawQuery() string           { return r.event.RawQueryString }
func (r *apiGatewayV2Request) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *apiGatewayV2Request) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *apiGatewayV2Request) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *apiGatewayV2Request) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}
func (r *apiGatewayV2Request) Unwrap() any { return r.event }

type functionURLRequest struct {
	event events.LambdaFunctionURLRequest
}

func (r *functionURLRequest) Method() string             { return r.event.RequestContext.HTTP.Method }
func (r *functionURLRequest) Path() string               { return r.event.RawPath }
func (r *functionURLRequest) RawQuery() string           { return r.event.RawQueryString }
func (r *functionURLRequest) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *functionURLRequest) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *functionURLRequest) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *functionURLRequest) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}
func (r *functionURLRequest) Unwrap() any { return r.event }

// UnwrapRequest returns the original event of type T behind req, and whether
// req actually wraps a T.
func UnwrapRequest[T any](req HttpRequest) (T, bool) {
	return unwrap[T](req.Unwrap())
}

func unwrap[T any](v any) (T, bool) {
	t, ok := v.(T)
	return t, ok
}

func decodeBody(body string, isBase64 bool) ([]byte, error) {
	if !isBase64 {
		return []byte(body), nil
	}
	return base64.StdEncoding.DecodeString(body)
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
