package humaserverless

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type echoInput struct {
	Name string `query:"name"`
	Body struct {
		Msg string `json:"msg"`
	}
}

type echoOutput struct {
	Body struct {
		Name string `json:"name"`
		Msg  string `json:"msg"`
	}
}

func newTestAPI(t *testing.T) huma.API {
	t.Helper()
	api := NewServerless(ApiOpts{Name: "test"})
	huma.Register(api, huma.Operation{
		OperationID: "echo",
		Method:      http.MethodPost,
		Path:        "/echo",
	}, func(ctx context.Context, in *echoInput) (*echoOutput, error) {
		out := &echoOutput{}
		out.Body.Name = in.Name
		out.Body.Msg = in.Body.Msg
		return out, nil
	})
	return api
}

func TestFunctionURLHandler(t *testing.T) {
	api := newTestAPI(t)

	resp, err := FunctionURLHandler(context.Background(), api, events.LambdaFunctionURLRequest{
		RawPath:        "/echo",
		RawQueryString: "name=bob",
		Headers:        map[string]string{"Content-Type": "application/json"},
		Body:           `{"msg":"hi"}`,
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: http.MethodPost},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"name":"bob","msg":"hi"}`, resp.Body)
}

func TestFunctionURLHandler_Base64Body(t *testing.T) {
	api := newTestAPI(t)

	resp, err := FunctionURLHandler(context.Background(), api, events.LambdaFunctionURLRequest{
		RawPath:         "/echo",
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            base64.StdEncoding.EncodeToString([]byte(`{"msg":"hi"}`)),
		IsBase64Encoded: true,
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: http.MethodPost},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"name":"","msg":"hi"}`, resp.Body)
}

func TestGenericHandler(t *testing.T) {
	api := newTestAPI(t)

	t.Run("api gateway v2", func(t *testing.T) {
		resp, err := Handler[events.APIGatewayV2HTTPRequest, events.APIGatewayV2HTTPResponse](
			context.Background(), api, events.APIGatewayV2HTTPRequest{
				RawPath:        "/echo",
				RawQueryString: "name=bob",
				Headers:        map[string]string{"Content-Type": "application/json"},
				Body:           `{"msg":"hi"}`,
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodPost},
				},
			})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.JSONEq(t, `{"name":"bob","msg":"hi"}`, resp.Body)
	})

	t.Run("function url", func(t *testing.T) {
		resp, err := Handler[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse](
			context.Background(), api, events.LambdaFunctionURLRequest{
				RawPath:        "/echo",
				RawQueryString: "name=bob",
				Headers:        map[string]string{"Content-Type": "application/json"},
				Body:           `{"msg":"hi"}`,
				RequestContext: events.LambdaFunctionURLRequestContext{
					HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: http.MethodPost},
				},
			})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.JSONEq(t, `{"name":"bob","msg":"hi"}`, resp.Body)
	})

	t.Run("mismatched types", func(t *testing.T) {
		_, err := Handler[events.LambdaFunctionURLRequest, events.APIGatewayV2HTTPResponse](
			context.Background(), api, events.LambdaFunctionURLRequest{})
		assert.Error(t, err)
	})
}

func TestHandlers_InvalidBase64BodyIs400(t *testing.T) {
	api := newTestAPI(t)

	url, err := FunctionURLHandler(context.Background(), api, events.LambdaFunctionURLRequest{
		RawPath: "/", Body: "!!", IsBase64Encoded: true,
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, url.StatusCode)

	v2, err := APIGatewayV2Handler(context.Background(), api, events.APIGatewayV2HTTPRequest{
		RawPath: "/", Body: "!!", IsBase64Encoded: true,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "POST"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, v2.StatusCode)
}
