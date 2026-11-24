package eventwrappers

import (
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// WrapAPIGatewayV2Request wraps an API Gateway v2 (HTTP API) request event.
func WrapAPIGatewayV2Request(event events.APIGatewayV2HTTPRequest) HttpRequest {
	return &apiGatewayV2Request{event: event}
}

// NewAPIGatewayV2Response wraps an API Gateway v2 response event. Pass nil for
// an empty 200 response; otherwise the wrapper mutates the given event.
func NewAPIGatewayV2Response(event *events.APIGatewayV2HTTPResponse) HttpResponse {
	if event == nil {
		event = &events.APIGatewayV2HTTPResponse{StatusCode: http.StatusOK}
	}
	return &apiGatewayV2Response{event: event}
}

type apiGatewayV2Request struct {
	event events.APIGatewayV2HTTPRequest
}

func (r *apiGatewayV2Request) Method() string { return r.event.RequestContext.HTTP.Method }
func (r *apiGatewayV2Request) Path() string {
	path := r.event.RawPath
	if stage := r.event.RequestContext.Stage; stage != "" {
		prefix := "/" + stage
		if path == prefix {
			return "/"
		}
		// Only strip a whole leading path segment, not a partial match
		if strings.HasPrefix(path, prefix+"/") {
			return path[len(prefix):]
		}
	}
	return path
}

func (r *apiGatewayV2Request) RawQuery() string           { return r.event.RawQueryString }
func (r *apiGatewayV2Request) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *apiGatewayV2Request) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *apiGatewayV2Request) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *apiGatewayV2Request) Unwrap() any                { return r.event }

func (r *apiGatewayV2Request) ToHttpRequest() (*http.Request, error) {
	return buildHttpRequest(r)
}
func (r *apiGatewayV2Request) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}

type apiGatewayV2Response struct {
	event *events.APIGatewayV2HTTPResponse
}

func (r *apiGatewayV2Response) StatusCode() int            { return r.event.StatusCode }
func (r *apiGatewayV2Response) SetStatusCode(code int)     { r.event.StatusCode = code }
func (r *apiGatewayV2Response) Headers() map[string]string { return copyHeaders(r.event.Headers) }
func (r *apiGatewayV2Response) Header(name string) string  { return lookupHeader(r.event.Headers, name) }
func (r *apiGatewayV2Response) Cookies() []string          { return append([]string(nil), r.event.Cookies...) }
func (r *apiGatewayV2Response) Unwrap() any                { return *r.event }
func (r *apiGatewayV2Response) Body() ([]byte, error) {
	return decodeBody(r.event.Body, r.event.IsBase64Encoded)
}

func (r *apiGatewayV2Response) SetBody(body string) {
	r.event.Body, r.event.IsBase64Encoded = body, false
}

func (r *apiGatewayV2Response) SetBinaryBody(body []byte) {
	r.event.Body, r.event.IsBase64Encoded = encodeBody(body), true
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

func (r *apiGatewayV2Response) FromHttpResponse(code int, header http.Header, body []byte) {
	r.event.StatusCode = code
	r.event.Cookies = nil
	r.event.Headers = map[string]string{}
	r.event.MultiValueHeaders = map[string][]string{}

	for name, values := range header {
		if len(values) > 0 {
			r.event.Headers[name] = values[len(values)-1]
			r.event.MultiValueHeaders[name] = values
		}
	}

	setHttpBody(r, r.event.Headers["Content-Type"], body)
}
