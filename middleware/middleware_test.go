package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/lucaspopp0/go-lambda/eventwrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chain(next HandlerFunc) HandlerFunc {
	return WithLogger(WithCORS(CORSOptions{ValidOrigins: []string{"https://ok.example"}})(next))
}

func okHandler(ctx context.Context, req eventwrappers.HttpRequest) (eventwrappers.HttpResponse, error) {
	resp := eventwrappers.NewResponseFor(req)
	resp.SetBody("hi " + req.Path())
	return resp, nil
}

func TestAdaptAPIGatewayV2(t *testing.T) {
	h := Adapt[events.APIGatewayV2HTTPRequest, events.APIGatewayV2HTTPResponse](chain(okHandler))

	out, err := h(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/foo",
		Headers: map[string]string{"Origin": "https://ok.example"},
	})
	require.NoError(t, err)
	assert.Equal(t, 200, out.StatusCode)
	assert.Equal(t, "hi /foo", out.Body)
	assert.Equal(t, "https://ok.example", out.Headers["Access-Control-Allow-Origin"])
}

func TestAdaptFunctionURL(t *testing.T) {
	h := Adapt[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse](chain(okHandler))

	out, err := h(context.Background(), events.LambdaFunctionURLRequest{
		RawPath: "/bar",
		Headers: map[string]string{"Origin": "https://evil.example"},
	})
	require.NoError(t, err)
	assert.Equal(t, "hi /bar", out.Body)
	assert.Equal(t, "true", out.Headers["Access-Control-Allow-Credentials"])
	assert.NotContains(t, out.Headers, "Access-Control-Allow-Origin")
}

func TestCORSConvertsErrorTo500(t *testing.T) {
	failing := func(ctx context.Context, req eventwrappers.HttpRequest) (eventwrappers.HttpResponse, error) {
		return nil, errors.New("boom")
	}
	h := Adapt[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse](chain(failing))

	out, err := h(context.Background(), events.LambdaFunctionURLRequest{})
	require.NoError(t, err)
	assert.Equal(t, 500, out.StatusCode)
	assert.Equal(t, "boom", out.Body)
}
