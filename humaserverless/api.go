package humaserverless

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

type ApiOpts struct {
	Name string

	Version string

	Servers []*huma.Server

	OAuth2 *OAuth2Opts
}

type OAuth2Opts struct {
	AuthDomain string
	Scopes     map[string]string

	AutoConfig *OAuth2AutoConfig
}

type OAuth2AutoConfig struct {
	ClientID string
	Scopes   []string
}

func NewServerless(opts ApiOpts) huma.API {
	if opts.Version == "" {
		opts.Version = "undefined"
	}

	router := chi.NewMux()

	config := huma.DefaultConfig(opts.Name, opts.Version)
	config.CreateHooks = nil

	api := humachi.New(router, config)

	api.OpenAPI().Servers = opts.Servers

	// Add OAuth 2.0 security scheme if Cognito domain is provided
	if opts.OAuth2 != nil {
		addOAuthSecurityScheme(api.OpenAPI(), opts)
	}

	return api
}

// addOAuthSecurityScheme adds OAuth 2.0 security scheme to the OpenAPI spec
func addOAuthSecurityScheme(spec *huma.OpenAPI, opts ApiOpts) {
	// Initialize Components if not already present
	if spec.Components == nil {
		spec.Components = &huma.Components{}
	}

	if spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = make(map[string]*huma.SecurityScheme)
	}

	if spec.Security == nil {
		spec.Security = []map[string][]string{}
	}

	if spec.Extensions == nil {
		spec.Extensions = map[string]any{}
	}

	schemeName := "OAuth2"

	authorizationURL := fmt.Sprintf("https://%s/authorize", opts.OAuth2.AuthDomain)
	tokenURL := fmt.Sprintf("https://%s/oauth2/token", opts.OAuth2.AuthDomain)

	scopeNames := slices.Collect(maps.Keys(opts.OAuth2.Scopes))

	spec.Components.SecuritySchemes[schemeName] = &huma.SecurityScheme{
		Type:        "oauth2",
		Name:        "Authorization",
		In:          "header",
		Scheme:      "OAuth",
		Description: "OAuth 2.0 Authorization Code flow with PKCE. Use AWS Cognito for authentication.",
		Flows: &huma.OAuthFlows{
			AuthorizationCode: &huma.OAuthFlow{
				AuthorizationURL: authorizationURL,
				TokenURL:         tokenURL,
				Scopes:           opts.OAuth2.Scopes,
			},
		},
	}

	spec.Security = append(spec.Security, map[string][]string{
		schemeName: scopeNames,
	})

	if opts.OAuth2.AutoConfig != nil {
		spec.Extensions["x-cli-config"] = huma.AutoConfig{
			Security: "OAuth2",
			Params: map[string]string{
				"client_id":     opts.OAuth2.AutoConfig.ClientID,
				"authorize_url": authorizationURL,
				"token_url":     tokenURL,
				"scopes":        strings.Join(opts.OAuth2.AutoConfig.Scopes, ","),
			},
		}
	}
}

// Request is the set of Lambda event types that can be served.
type Request interface {
	events.APIGatewayV2HTTPRequest | events.LambdaFunctionURLRequest
}

// Response is the set of Lambda response types matching Request.
type Response interface {
	events.APIGatewayV2HTTPResponse | events.LambdaFunctionURLResponse
}

// Handler handles a single event of either supported type using a pre-built API
// (caller must register routes). The response type must match the request type:
// APIGatewayV2HTTPRequest -> APIGatewayV2HTTPResponse,
// LambdaFunctionURLRequest -> LambdaFunctionURLResponse.
func Handler[Req Request, Resp Response](
	ctx context.Context,
	api huma.API,
	event Req,
) (Resp, error) {
	var resp Resp

	switch e := any(event).(type) {
	case events.APIGatewayV2HTTPRequest:
		out, ok := any(&resp).(*events.APIGatewayV2HTTPResponse)
		if !ok {
			return resp, fmt.Errorf("humaserverless: response type %T does not match request type %T", resp, event)
		}
		*out, _ = APIGatewayV2Handler(ctx, api, e)
	case events.LambdaFunctionURLRequest:
		out, ok := any(&resp).(*events.LambdaFunctionURLResponse)
		if !ok {
			return resp, fmt.Errorf("humaserverless: response type %T does not match request type %T", resp, event)
		}
		*out, _ = FunctionURLHandler(ctx, api, e)
	}

	return resp, nil
}

// HttpHandler is the original name of APIGatewayV2Handler.
//
// Deprecated: use APIGatewayV2Handler.
func HttpHandler(
	ctx context.Context,
	api huma.API,
	event events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {
	return APIGatewayV2Handler(ctx, api, event)
}

// APIGatewayV2Handler handles a single API Gateway v2 (HTTP API) request using a
// pre-built API (caller must register routes).
func APIGatewayV2Handler(
	ctx context.Context,
	api huma.API,
	event events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {
	request, err := EventToRequest(event)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}
	response := httptest.NewRecorder()
	api.Adapter().ServeHTTP(response, request.WithContext(ctx))
	return *ResponseToEvent(*response), nil
}

// FunctionURLHandler handles a single Lambda function URL request using a pre-built API
// (caller must register routes).
func FunctionURLHandler(
	ctx context.Context,
	api huma.API,
	event events.LambdaFunctionURLRequest,
) (events.LambdaFunctionURLResponse, error) {
	request, err := FunctionURLEventToRequest(event)
	if err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}
	response := httptest.NewRecorder()
	api.Adapter().ServeHTTP(response, request.WithContext(ctx))
	return *ResponseToFunctionURLEvent(*response), nil
}
