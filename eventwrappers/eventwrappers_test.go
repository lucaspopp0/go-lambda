package eventwrappers

import (
	"encoding/base64"
	"io"
	"net/http"
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

func TestToHttpRequest(t *testing.T) {
	t.Run("apigw-v2 strips stage and keeps query, headers, cookies", func(t *testing.T) {
		event := events.APIGatewayV2HTTPRequest{
			RawPath: "/prod/users", RawQueryString: "a=1",
			Headers: map[string]string{"X-Test": "v"}, Cookies: []string{"a=1", "b=2"},
			Body: base64.StdEncoding.EncodeToString([]byte("hi")), IsBase64Encoded: true,
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				Stage: "prod",
				HTTP:  events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "POST"},
			},
		}
		r, err := WrapAPIGatewayV2Request(event).ToHttpRequest()
		require.NoError(t, err)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/users", r.URL.Path)
		assert.Equal(t, "a=1", r.URL.RawQuery)
		assert.Equal(t, "v", r.Header.Get("X-Test"))
		assert.Equal(t, "a=1; b=2", r.Header.Get("Cookie"))
		b, _ := io.ReadAll(r.Body)
		assert.Equal(t, "hi", string(b))
	})

	t.Run("function-url does not strip a stage-like prefix", func(t *testing.T) {
		r, err := WrapFunctionURLRequest(events.LambdaFunctionURLRequest{RawPath: "/prod/users"}).ToHttpRequest()
		require.NoError(t, err)
		assert.Equal(t, "/prod/users", r.URL.Path)
	})

	t.Run("invalid base64 errors", func(t *testing.T) {
		_, err := WrapFunctionURLRequest(events.LambdaFunctionURLRequest{Body: "!!", IsBase64Encoded: true}).ToHttpRequest()
		assert.Error(t, err)
	})
}

func fromHTTP[T any](resp HttpResponse, code int, h http.Header, body []byte) (T, bool) {
	resp.FromHttpResponse(code, h, body)
	return UnwrapResponse[T](resp)
}

func TestFromHttpResponse(t *testing.T) {
	h := http.Header{}
	h.Add("Content-Type", "image/png")
	h.Add("X-Multi", "a")
	h.Add("X-Multi", "b")
	h.Add("Set-Cookie", "a=1")
	h.Add("Set-Cookie", "b=2")

	t.Run("apigw-v2", func(t *testing.T) {
		e, ok := fromHTTP[events.APIGatewayV2HTTPResponse](NewAPIGatewayV2Response(nil), 201, h, []byte{1, 2})
		require.True(t, ok)
		assert.Equal(t, 201, e.StatusCode)
		assert.Equal(t, "b", e.Headers["X-Multi"])
		assert.Equal(t, []string{"a", "b"}, e.MultiValueHeaders["X-Multi"])
		assert.True(t, e.IsBase64Encoded)
	})

	t.Run("function-url", func(t *testing.T) {
		e, ok := fromHTTP[events.LambdaFunctionURLResponse](NewFunctionURLResponse(nil), 200, h, []byte{1, 2})
		require.True(t, ok)
		assert.Equal(t, "a, b", e.Headers["X-Multi"])
		assert.Equal(t, []string{"a=1", "b=2"}, e.Cookies)
		assert.NotContains(t, e.Headers, "Set-Cookie")
		assert.True(t, e.IsBase64Encoded)
	})

	t.Run("text body", func(t *testing.T) {
		e, _ := fromHTTP[events.LambdaFunctionURLResponse](NewFunctionURLResponse(nil), 200, http.Header{}, []byte("hi"))
		assert.Equal(t, "hi", e.Body)
		assert.False(t, e.IsBase64Encoded)
	})
}

func TestGenericConversions(t *testing.T) {
	r, err := ToHttpRequest(events.APIGatewayV2HTTPRequest{
		RawPath: "/prod/x", RawQueryString: "q=1",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Stage: "prod",
			HTTP:  events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "/x", r.URL.Path)
	assert.Equal(t, "q=1", r.URL.RawQuery)

	r, err = ToHttpRequest(events.LambdaFunctionURLRequest{RawPath: "/prod/x"})
	require.NoError(t, err)
	assert.Equal(t, "/prod/x", r.URL.Path)

	h := http.Header{}
	h.Add("X-Multi", "a")
	h.Add("X-Multi", "b")
	v2 := FromHttpResponse[events.APIGatewayV2HTTPResponse](201, h, []byte("hi"))
	assert.Equal(t, 201, v2.StatusCode)
	assert.Equal(t, "b", v2.Headers["X-Multi"])
	assert.Equal(t, "hi", v2.Body)

	url := FromHttpResponse[events.LambdaFunctionURLResponse](200, h, nil)
	assert.Equal(t, "a, b", url.Headers["X-Multi"])
}

func TestWrapRequestAndResponse(t *testing.T) {
	req := WrapRequest(events.LambdaFunctionURLRequest{RawPath: "/a"})
	assert.Equal(t, "/a", req.Path())
	_, ok := UnwrapRequest[events.LambdaFunctionURLRequest](req)
	assert.True(t, ok)

	event := &events.APIGatewayV2HTTPResponse{StatusCode: 200}
	resp := WrapResponse(event)
	resp.SetStatusCode(418)
	assert.Equal(t, 418, event.StatusCode)

	empty := WrapResponse[events.LambdaFunctionURLResponse](nil)
	assert.Equal(t, 200, empty.StatusCode())
	_, ok = UnwrapResponse[events.LambdaFunctionURLResponse](empty)
	assert.True(t, ok)
}

func TestAPIGatewayV2PathStripsWholeStageSegment(t *testing.T) {
	cases := map[string]string{
		"/prod/users": "/users",
		"/prod":       "/",
		"/prod/":      "/",
		"/products/1": "/products/1",
		"/users":      "/users",
	}
	for raw, want := range cases {
		req := WrapAPIGatewayV2Request(events.APIGatewayV2HTTPRequest{
			RawPath:        raw,
			RequestContext: events.APIGatewayV2HTTPRequestContext{Stage: "prod"},
		})
		assert.Equal(t, want, req.Path(), raw)
	}
}
