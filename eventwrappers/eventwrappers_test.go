package eventwrappers

import (
	"encoding/base64"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requestFixtures() map[string]HttpRequest {
	headers := map[string]string{"content-type": "application/json"}
	return map[string]HttpRequest{
		"apigw-v2": WrapAPIGatewayV2Request(events.APIGatewayV2HTTPRequest{
			RawPath: "/users", RawQueryString: "a=1", Headers: headers,
			Cookies: []string{"x=1"}, Body: "hi",
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "POST"},
			},
		}),
		"function-url": WrapFunctionURLRequest(events.LambdaFunctionURLRequest{
			RawPath: "/users", RawQueryString: "a=1", Headers: headers,
			Cookies: []string{"x=1"}, Body: "hi",
			RequestContext: events.LambdaFunctionURLRequestContext{
				HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
			},
		}),
	}
}

func TestHttpRequest(t *testing.T) {
	for name, req := range requestFixtures() {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, "POST", req.Method())
			assert.Equal(t, "/users", req.Path())
			assert.Equal(t, "a=1", req.RawQuery())
			assert.Equal(t, "application/json", req.Header("Content-Type"))
			assert.Equal(t, []string{"x=1"}, req.Cookies())
			body, err := req.Body()
			require.NoError(t, err)
			assert.Equal(t, "hi", string(body))

			req.Headers()["content-type"] = "mutated"
			assert.Equal(t, "application/json", req.Header("content-type"))
		})
	}

	t.Run("decodes-base64-body", func(t *testing.T) {
		req := WrapFunctionURLRequest(events.LambdaFunctionURLRequest{
			Body: base64.StdEncoding.EncodeToString([]byte{0, 1, 2}), IsBase64Encoded: true,
		})
		body, err := req.Body()
		require.NoError(t, err)
		assert.Equal(t, []byte{0, 1, 2}, body)
	})

	t.Run("invalid-base64-errors", func(t *testing.T) {
		req := WrapAPIGatewayV2Request(events.APIGatewayV2HTTPRequest{Body: "!!", IsBase64Encoded: true})
		_, err := req.Body()
		assert.Error(t, err)
	})
}

func TestUnwrapRequest(t *testing.T) {
	reqs := requestFixtures()

	v2, ok := UnwrapRequest[events.APIGatewayV2HTTPRequest](reqs["apigw-v2"])
	require.True(t, ok)
	assert.Equal(t, "/users", v2.RawPath)

	_, ok = UnwrapRequest[events.LambdaFunctionURLRequest](reqs["apigw-v2"])
	assert.False(t, ok)

	url, ok := UnwrapRequest[events.LambdaFunctionURLRequest](reqs["function-url"])
	require.True(t, ok)
	assert.Equal(t, "/users", url.RawPath)
	assert.IsType(t, events.LambdaFunctionURLRequest{}, reqs["function-url"].Unwrap())
}

func responseFixtures() map[string]HttpResponse {
	return map[string]HttpResponse{
		"apigw-v2":     NewAPIGatewayV2Response(nil),
		"function-url": NewFunctionURLResponse(nil),
	}
}

func TestHttpResponse(t *testing.T) {
	for name, resp := range responseFixtures() {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 200, resp.StatusCode())
			resp.SetStatusCode(201)
			assert.Equal(t, 201, resp.StatusCode())

			resp.SetHeader("Content-Type", "text/plain")
			resp.SetHeader("content-type", "application/json")
			assert.Equal(t, map[string]string{"content-type": "application/json"}, resp.Headers())

			resp.AddHeader("X-Multi", "a")
			resp.AddHeader("x-multi", "b")
			assert.Equal(t, "a, b", resp.Header("X-Multi"))

			resp.AddHeader("Set-Cookie", "a=1")
			resp.AddHeader("set-cookie", "b=2")
			assert.Equal(t, []string{"a=1", "b=2"}, resp.Cookies())
			assert.NotContains(t, resp.Headers(), "Set-Cookie")

			resp.SetBody("hello")
			body, err := resp.Body()
			require.NoError(t, err)
			assert.Equal(t, "hello", string(body))

			resp.SetBinaryBody([]byte{9, 8})
			body, err = resp.Body()
			require.NoError(t, err)
			assert.Equal(t, []byte{9, 8}, body)
		})
	}
}

func TestUnwrapResponse(t *testing.T) {
	resps := responseFixtures()
	resps["apigw-v2"].SetStatusCode(404)
	resps["function-url"].SetBinaryBody([]byte("x"))

	v2, ok := UnwrapResponse[events.APIGatewayV2HTTPResponse](resps["apigw-v2"])
	require.True(t, ok)
	assert.Equal(t, 404, v2.StatusCode)

	_, ok = UnwrapResponse[events.LambdaFunctionURLResponse](resps["apigw-v2"])
	assert.False(t, ok)

	url, ok := UnwrapResponse[events.LambdaFunctionURLResponse](resps["function-url"])
	require.True(t, ok)
	assert.True(t, url.IsBase64Encoded)
}

func TestWrapperMutatesProvidedEvent(t *testing.T) {
	event := &events.LambdaFunctionURLResponse{StatusCode: 200}
	NewFunctionURLResponse(event).SetStatusCode(500)
	assert.Equal(t, 500, event.StatusCode)
}
