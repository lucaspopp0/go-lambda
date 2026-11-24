package humaserverless

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/lucaspopp0/go-lambda/eventwrappers"
)

// Convert the API Gateway event into a HTTP request
func EventToRequest(event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	requestURL := event.RawPath
	if event.RequestContext.Stage != "" {
		requestURL = strings.TrimPrefix(requestURL,
			fmt.Sprintf("/%s", event.RequestContext.Stage))
	}

	return buildRequest(eventwrappers.WrapAPIGatewayV2Request(event), requestURL)
}

// Convert the Lambda function URL event into a HTTP request
func FunctionURLEventToRequest(event events.LambdaFunctionURLRequest) (*http.Request, error) {
	req := eventwrappers.WrapFunctionURLRequest(event)
	return buildRequest(req, req.Path())
}

// buildRequest converts any wrapped Lambda HTTP event into an http.Request.
// path overrides req.Path() so callers can adjust it (e.g. strip a stage prefix).
func buildRequest(req eventwrappers.HttpRequest, path string) (*http.Request, error) {
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

	for headerName, headerVal := range req.Headers() {
		request.Header.Set(headerName, headerVal)
	}

	if cookies := req.Cookies(); len(cookies) > 0 {
		request.Header.Set("Cookie", strings.Join(cookies, "; "))
	}

	return request, nil
}

// Serve the HTTP request to the API's adapter
// Convert the response to the API gateway response
func ResponseToEvent(response httptest.ResponseRecorder) *events.APIGatewayV2HTTPResponse {
	event := &events.APIGatewayV2HTTPResponse{
		StatusCode:        response.Code,
		Headers:           map[string]string{},
		MultiValueHeaders: map[string][]string{},
	}

	for headerName, headerValues := range response.Header() {
		if len(headerValues) > 0 {
			event.Headers[headerName] = headerValues[len(headerValues)-1]
			event.MultiValueHeaders[headerName] = headerValues
		}
	}

	contentType := event.Headers["Content-Type"]
	if isBinaryContentType(contentType) {
		event.Body = base64.StdEncoding.EncodeToString(response.Body.Bytes())
		event.IsBase64Encoded = true
	} else {
		event.Body = response.Body.String()
	}

	return event
}

// Convert the recorded response to a Lambda function URL response
func ResponseToFunctionURLEvent(response httptest.ResponseRecorder) *events.LambdaFunctionURLResponse {
	event := &events.LambdaFunctionURLResponse{
		StatusCode: response.Code,
		Headers:    map[string]string{},
	}

	for headerName, headerValues := range response.Header() {
		if len(headerValues) == 0 {
			continue
		}

		if http.CanonicalHeaderKey(headerName) == "Set-Cookie" {
			event.Cookies = append(event.Cookies, headerValues...)
			continue
		}

		// Function URLs only support single-valued headers; join multiple values
		event.Headers[headerName] = strings.Join(headerValues, ", ")
	}

	if isBinaryContentType(event.Headers["Content-Type"]) {
		event.Body = base64.StdEncoding.EncodeToString(response.Body.Bytes())
		event.IsBase64Encoded = true
	} else {
		event.Body = response.Body.String()
	}

	return event
}

func isBinaryContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") ||
		contentType == "application/octet-stream"
}
