package middleware

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// Request is the set of Lambda event types the middleware supports.
type Request interface {
	events.APIGatewayV2HTTPRequest | events.LambdaFunctionURLRequest
}

// Response is the set of Lambda response types matching Request.
type Response interface {
	events.APIGatewayV2HTTPResponse | events.LambdaFunctionURLResponse
}

// HandlerFunc is a Lambda handler for either supported event type. The response
// type must match the request type: APIGatewayV2HTTPRequest pairs with
// APIGatewayV2HTTPResponse, and LambdaFunctionURLRequest with
// LambdaFunctionURLResponse.
type HandlerFunc[Req Request, Resp Response] func(ctx context.Context, event Req) (Resp, error)

// APIGatewayV2HandlerFunc is a HandlerFunc for API Gateway v2 (HTTP API) events.
type APIGatewayV2HandlerFunc = HandlerFunc[events.APIGatewayV2HTTPRequest, events.APIGatewayV2HTTPResponse]

// FunctionURLHandlerFunc is a HandlerFunc for Lambda Function URL events.
type FunctionURLHandlerFunc = HandlerFunc[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse]

// requestHeaders returns the headers of the event.
func requestHeaders[Req Request](event Req) map[string]string {
	switch e := any(event).(type) {
	case events.APIGatewayV2HTTPRequest:
		return e.Headers
	case events.LambdaFunctionURLRequest:
		return e.Headers
	}
	return nil
}

// statusCode returns the status code of the response.
func statusCode[Resp Response](response Resp) int {
	switch r := any(response).(type) {
	case events.APIGatewayV2HTTPResponse:
		return r.StatusCode
	case events.LambdaFunctionURLResponse:
		return r.StatusCode
	}
	return 0
}

// setHeader sets a header on the response, allocating the header map if needed.
func setHeader[Resp Response](response *Resp, name, value string) {
	switch r := any(response).(type) {
	case *events.APIGatewayV2HTTPResponse:
		if r.Headers == nil {
			r.Headers = make(map[string]string)
		}
		r.Headers[name] = value
	case *events.LambdaFunctionURLResponse:
		if r.Headers == nil {
			r.Headers = make(map[string]string)
		}
		r.Headers[name] = value
	}
}

// errorResponse builds a response with the given status code and body.
func errorResponse[Resp Response](status int, body string) Resp {
	var response Resp
	switch r := any(&response).(type) {
	case *events.APIGatewayV2HTTPResponse:
		r.StatusCode, r.Body = status, body
	case *events.LambdaFunctionURLResponse:
		r.StatusCode, r.Body = status, body
	}
	return response
}
