package eventwrappers

import (
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// WrapFunctionURLRequest wraps a Lambda Function URL request event.
func WrapFunctionURLRequest(event events.LambdaFunctionURLRequest) HttpRequest {
	return &functionURLRequest{event: event}
}

// NewFunctionURLResponse wraps a Lambda Function URL response event. Pass nil
// for an empty 200 response; otherwise the wrapper mutates the given event.
func NewFunctionURLResponse(event *events.LambdaFunctionURLResponse) HttpResponse {
	if event == nil {
		event = &events.LambdaFunctionURLResponse{StatusCode: http.StatusOK}
	}
	return &functionURLResponse{event: event}
}

type functionURLRequest struct {
	event events.LambdaFunctionURLRequest
}

func (r *functionURLRequest) Method() string             { return r.event.RequestContext.HTTP.Method }
func (r *functionURLRequest) Path() string               { return r.event.RawPath }
func (r *functionURLRequest) RawQuery() string           { return r.event.RawQueryString }
func (r *functionURLRequest) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *functionURLRequest) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *functionURLRequest) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *functionURLRequest) Unwrap() any                { return r.event }

func (r *functionURLRequest) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}

func (r *functionURLRequest) ToHttpRequest() (*http.Request, error) {
	return buildHttpRequest(r)
}

type functionURLResponse struct {
	event *events.LambdaFunctionURLResponse
}

func (r *functionURLResponse) StatusCode() int            { return r.event.StatusCode }
func (r *functionURLResponse) SetStatusCode(code int)     { r.event.StatusCode = code }
func (r *functionURLResponse) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *functionURLResponse) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *functionURLResponse) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *functionURLResponse) Unwrap() any                { return *r.event }
func (r *functionURLResponse) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}

func (r *functionURLResponse) SetBody(body string) {
	r.event.Body, r.event.IsBase64Encoded = body, false
}

func (r *functionURLResponse) SetBinaryBody(body []byte) {
	r.event.Body, r.event.IsBase64Encoded = encodeBody(body), true
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

func (r *functionURLResponse) FromHttpResponse(code int, header http.Header, body []byte) {
	r.event.StatusCode = code
	r.event.Cookies = nil
	r.event.Headers = map[string]string{}

	for name, values := range header {
		if len(values) == 0 {
			continue
		}
		if http.CanonicalHeaderKey(name) == "Set-Cookie" {
			r.event.Cookies = append(r.event.Cookies, values...)
			continue
		}
		// Function URLs only support single-valued headers; join multiple values
		r.event.Headers[name] = strings.Join(values, ", ")
	}

	setHttpBody(r, r.event.Headers["Content-Type"], body)
}
