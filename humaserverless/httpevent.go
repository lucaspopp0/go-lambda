package humaserverless

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Convert the API Gateway event into a HTTP request
func EventToRequest(event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	requestURL := event.RawPath
	if event.RequestContext.Stage != "" {
		requestURL = strings.TrimPrefix(requestURL,
			fmt.Sprintf("/%s", event.RequestContext.Stage))
	}

	return buildRequest(
		event.RequestContext.HTTP.Method,
		requestURL,
		event.RawQueryString,
		event.Body,
		event.IsBase64Encoded,
		event.Headers,
		event.Cookies,
	)
}

// Convert the Lambda function URL event into a HTTP request
func FunctionURLEventToRequest(event events.LambdaFunctionURLRequest) (*http.Request, error) {
	return buildRequest(
		event.RequestContext.HTTP.Method,
		event.RawPath,
		event.RawQueryString,
		event.Body,
		event.IsBase64Encoded,
		event.Headers,
		event.Cookies,
	)
}

func buildRequest(
	method, path, rawQuery, body string,
	isBase64 bool,
	headers map[string]string,
	cookies []string,
) (*http.Request, error) {
	var bodyReader io.Reader = strings.NewReader(body)
	if isBase64 {
		bodyReader = base64.NewDecoder(base64.StdEncoding, bodyReader)
	}

	requestURL := path
	if rawQuery != "" {
		requestURL = fmt.Sprintf("%s?%s", requestURL, rawQuery)
	}

	request, err := http.NewRequest(method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}

	for headerName, headerVal := range headers {
		request.Header.Set(headerName, headerVal)
	}

	if len(cookies) > 0 {
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
