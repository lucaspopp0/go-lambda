package humaserverless

import (
	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-lambda-go/events"
	"github.com/lucaspopp0/go-lambda/eventwrappers"
)

// Convert the API Gateway event into a HTTP request
func EventToRequest(event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	return eventwrappers.ToHttpRequest(event)
}

// Convert the Lambda function URL event into a HTTP request
func FunctionURLEventToRequest(event events.LambdaFunctionURLRequest) (*http.Request, error) {
	return eventwrappers.ToHttpRequest(event)
}

// Serve the HTTP request to the API's adapter
// Convert the response to the API gateway response
func ResponseToEvent(response httptest.ResponseRecorder) *events.APIGatewayV2HTTPResponse {
	event := eventwrappers.FromHttpResponse[events.APIGatewayV2HTTPResponse](response.Code, response.Header(), response.Body.Bytes())
	return &event
}

// Convert the recorded response to a Lambda function URL response
func ResponseToFunctionURLEvent(response httptest.ResponseRecorder) *events.LambdaFunctionURLResponse {
	event := eventwrappers.FromHttpResponse[events.LambdaFunctionURLResponse](response.Code, response.Header(), response.Body.Bytes())
	return &event
}
