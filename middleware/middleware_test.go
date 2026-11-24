package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCORS_APIGatewayV2(t *testing.T) {
	h := WithCORS[events.APIGatewayV2HTTPRequest, events.APIGatewayV2HTTPResponse](
		CORSOptions{ValidOrigins: []string{"https://a.example"}},
	)(func(ctx context.Context, e events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return events.APIGatewayV2HTTPResponse{StatusCode: 200}, nil
	})

	resp, err := h(context.Background(), events.APIGatewayV2HTTPRequest{Headers: map[string]string{"origin": "https://a.example"}})
	require.NoError(t, err)
	assert.Equal(t, "https://a.example", resp.Headers["Access-Control-Allow-Origin"])
	assert.Equal(t, "true", resp.Headers["Access-Control-Allow-Credentials"])

	resp, _ = h(context.Background(), events.APIGatewayV2HTTPRequest{Headers: map[string]string{"Origin": "https://evil.example"}})
	assert.NotContains(t, resp.Headers, "Access-Control-Allow-Origin")
}

func TestCORS_FunctionURL(t *testing.T) {
	opts := CORSOptions{ValidOrigins: []string{"https://a.example"}}
	h := WithCORS[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse](opts)(
		func(ctx context.Context, e events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
			return events.LambdaFunctionURLResponse{StatusCode: 200}, nil
		})

	resp, err := h(context.Background(), events.LambdaFunctionURLRequest{Headers: map[string]string{"origin": "https://a.example"}})
	require.NoError(t, err)
	assert.Equal(t, "https://a.example", resp.Headers["Access-Control-Allow-Origin"])
	assert.Equal(t, "*", resp.Headers["Access-Control-Allow-Methods"])

	failing := WithCORS[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse](opts)(
		func(ctx context.Context, e events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
			return events.LambdaFunctionURLResponse{}, errors.New("boom")
		})
	resp, err = failing(context.Background(), events.LambdaFunctionURLRequest{})
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, "boom", resp.Body)
}

func TestDefaultChain_BothEventTypes(t *testing.T) {
	var gotLogger bool
	apigw := DefaultChain(func(ctx context.Context, e events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		_, gotLogger = ctx.Value(ctxKeyLogger).(interface{})
		return events.APIGatewayV2HTTPResponse{StatusCode: 404}, nil
	})
	resp, err := apigw(context.Background(), events.APIGatewayV2HTTPRequest{Headers: map[string]string{"origin": "http://localhost:8081"}})
	require.NoError(t, err)
	assert.True(t, gotLogger)
	assert.Equal(t, "http://localhost:8081", resp.Headers["Access-Control-Allow-Origin"])

	furl := DefaultChain(func(ctx context.Context, e events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		return events.LambdaFunctionURLResponse{StatusCode: 503}, nil
	})
	fresp, err := furl(context.Background(), events.LambdaFunctionURLRequest{Headers: map[string]string{"origin": "http://localhost:8081"}})
	require.NoError(t, err)
	assert.Equal(t, 503, fresp.StatusCode)
	assert.Equal(t, "http://localhost:8081", fresp.Headers["Access-Control-Allow-Origin"])
}
