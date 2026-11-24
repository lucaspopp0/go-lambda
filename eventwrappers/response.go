package eventwrappers

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

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
	// Unwrap returns the original event value (e.g. events.APIGatewayV2HTTPResponse).
	Unwrap() any
}

// NewAPIGatewayV2Response wraps an API Gateway v2 response event. Pass nil for
// an empty response; otherwise the wrapper mutates the given event.
func NewAPIGatewayV2Response(event *events.APIGatewayV2HTTPResponse) HttpResponse {
	if event == nil {
		event = &events.APIGatewayV2HTTPResponse{StatusCode: http.StatusOK}
	}
	return &apiGatewayV2Response{event: event}
}

// NewFunctionURLResponse wraps a Lambda Function URL response event. Pass nil
// for an empty response; otherwise the wrapper mutates the given event.
func NewFunctionURLResponse(event *events.LambdaFunctionURLResponse) HttpResponse {
	if event == nil {
		event = &events.LambdaFunctionURLResponse{StatusCode: http.StatusOK}
	}
	return &functionURLResponse{event: event}
}

type apiGatewayV2Response struct {
	event *events.APIGatewayV2HTTPResponse
}

func (r *apiGatewayV2Response) StatusCode() int            { return r.event.StatusCode }
func (r *apiGatewayV2Response) SetStatusCode(code int)     { r.event.StatusCode = code }
func (r *apiGatewayV2Response) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *apiGatewayV2Response) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *apiGatewayV2Response) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *apiGatewayV2Response) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}
func (r *apiGatewayV2Response) SetBody(body string) {
	r.event.Body, r.event.IsBase64Encoded = body, false
}
func (r *apiGatewayV2Response) SetBinaryBody(body []byte) { r.setBinary(body) }
func (r *apiGatewayV2Response) Unwrap() any               { return *r.event }
func (r *apiGatewayV2Response) setBinary(body []byte) {
	r.event.Body, r.event.IsBase64Encoded = base64.StdEncoding.EncodeToString(body), true
}

func (r *apiGatewayV2Response) SetHeader(name, value string) {
	if isSetCookie(name) {
		r.event.Cookies = []string{value}
		return
	}
	deleteHeader(r.event.Headers, name)
	deleteHeader(r.event.MultiValueHeaders, name)
	if r.event.Headers == nil {
		r.event.Headers = map[string]string{}
	}
	r.event.Headers[name] = value
	if r.event.MultiValueHeaders != nil {
		r.event.MultiValueHeaders[name] = []string{value}
	}
}

func (r *apiGatewayV2Response) AddHeader(name, value string) {
	if isSetCookie(name) {
		r.event.Cookies = append(r.event.Cookies, value)
		return
	}
	if existing := lookupHeader(r.event.Headers, name); existing != "" {
		value = existing + ", " + value
	}
	r.SetHeader(name, value)
}

type functionURLResponse struct {
	event *events.LambdaFunctionURLResponse
}

func (r *functionURLResponse) StatusCode() int            { return r.event.StatusCode }
func (r *functionURLResponse) SetStatusCode(code int)     { r.event.StatusCode = code }
func (r *functionURLResponse) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *functionURLResponse) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *functionURLResponse) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *functionURLResponse) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}
func (r *functionURLResponse) SetBody(body string) {
	r.event.Body, r.event.IsBase64Encoded = body, false
}
func (r *functionURLResponse) Unwrap() any { return *r.event }
func (r *functionURLResponse) SetBinaryBody(body []byte) {
	r.event.Body, r.event.IsBase64Encoded = base64.StdEncoding.EncodeToString(body), true
}

func (r *functionURLResponse) SetHeader(name, value string) {
	if isSetCookie(name) {
		r.event.Cookies = []string{value}
		return
	}
	deleteHeader(r.event.Headers, name)
	if r.event.Headers == nil {
		r.event.Headers = map[string]string{}
	}
	r.event.Headers[name] = value
}

func (r *functionURLResponse) AddHeader(name, value string) {
	if isSetCookie(name) {
		r.event.Cookies = append(r.event.Cookies, value)
		return
	}
	if existing := lookupHeader(r.event.Headers, name); existing != "" {
		value = existing + ", " + value
	}
	r.SetHeader(name, value)
}

// UnwrapResponse returns the original event of type T behind resp, and whether
// resp actually wraps a T.
func UnwrapResponse[T any](resp HttpResponse) (T, bool) {
	return unwrap[T](resp.Unwrap())
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
