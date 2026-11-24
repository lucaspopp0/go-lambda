# go-lambda

Go modules for building AWS Lambda functions: structured logging, HTTP middleware, and a bridge for serving [huma](https://huma.rocks) APIs from Lambda.

```bash
go get github.com/lucaspopp0/go-lambda
```

| Package | Import path | Purpose |
| --- | --- | --- |
| [`humaserverless`](#humaserverless) | `github.com/lucaspopp0/go-lambda/humaserverless` | Serve a huma API from API Gateway v2 or Lambda Function URL events |
| [`middleware`](#middleware) | `github.com/lucaspopp0/go-lambda/middleware` | Composable handler middleware (logging, CORS) for API Gateway v2 |
| [`eventwrappers`](#eventwrappers) | `github.com/lucaspopp0/go-lambda/eventwrappers` | Common `HttpRequest`/`HttpResponse` interfaces over API Gateway v2 and Function URL events |
| [`logs`](#logs) | `github.com/lucaspopp0/go-lambda/logs` | Context-aware structured logger built on `log/slog` |

## humaserverless

Serve a [huma](https://huma.rocks) API from Lambda. Supports API Gateway v2 (HTTP API) and Lambda Function URL events.

```go
api := humaserverless.NewServerless(humaserverless.ApiOpts{Name: "my-api"})
// huma.Register(api, ...)

// API Gateway v2
lambda.Start(func(ctx context.Context, e events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return humaserverless.APIGatewayV2Handler(ctx, api, e)
})

// Lambda Function URL
lambda.Start(func(ctx context.Context, e events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return humaserverless.FunctionURLHandler(ctx, api, e)
})

// Or generically:
// lambda.Start(func(ctx context.Context, e events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
//	return humaserverless.Handler[events.LambdaFunctionURLRequest, events.LambdaFunctionURLResponse](ctx, api, e)
// })
```

## eventwrappers

`HttpRequest` and `HttpResponse` interfaces that abstract over API Gateway v2 and Lambda Function URL events, so handler logic can be written once. Each wrapper can be unwrapped to the original event type.

```go
req := eventwrappers.WrapFunctionURLRequest(event)
resp := eventwrappers.NewFunctionURLResponse(nil)
resp.SetStatusCode(200)
resp.SetHeader("Content-Type", "application/json")
resp.SetBody(`{"path":"` + req.Path() + `"}`)

// Convert to / from net/http
httpReq, _ := eventwrappers.ToHTTPRequest(req)
resp = eventwrappers.FunctionURLResponseFromHTTP(code, header, body) // or APIGatewayV2ResponseFromHTTP

// Back to the original types
original, ok := eventwrappers.UnwrapRequest[events.LambdaFunctionURLRequest](req)
out, _ := eventwrappers.UnwrapResponse[events.LambdaFunctionURLResponse](resp)
```

## middleware

Handler middleware for API Gateway v2 (HTTP API) Lambda handlers. Middleware wraps a `middleware.HandlerFunc`, which has the signature `func(ctx, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)`.

- `WithLogger` creates a `logs.Logger`, stores it in the context, and logs each request and response with the event attached as annotations. Responses are logged at error level for 5xx, warn for 4xx, and info otherwise.
- `GetLogger(ctx)` retrieves that logger, falling back to a default stdout logger (with a warning) if none is set.
- `WithCORS(CORSOptions{ValidOrigins: ...})` adds CORS headers and echoes the request `Origin` only if it is in `ValidOrigins`. Errors returned by the wrapped handler are logged and turned into a 500 response.
- `DefaultChain` wraps a handler in `WithLogger` and `WithCORS`. It only allows the origin `http://localhost:8081`, so build your own chain for other origins.

```go
handler := middleware.WithLogger(
	middleware.WithCORS(middleware.CORSOptions{
		ValidOrigins: []string{"https://app.example.com"},
	})(myHandler),
)

lambda.Start(handler)
```

`middleware` currently supports only API Gateway v2 events, not Lambda Function URL events.

## logs

A small structured logger on top of `log/slog` that attaches annotations stored in the `context.Context` to every log line.

- `logs.NewLogger(logs.LoggerOpts{Out, Style, Level})` returns a `Logger` with `Info`, `Warn` and `Error` methods, each with an `f` variant (`Infof`, ...). `Style` is `StyleJSON` (the default) or `StyleText`, and `Level` defaults to info.
- `logs.WithAnnotation(ctx, key, value)` and `logs.WithAnnotations(ctx, map)` return a context carrying extra fields. `logs.GetAnnotations(ctx)` reads them back.

```go
logger := logs.NewLogger(logs.LoggerOpts{Out: os.Stdout})

ctx = logs.WithAnnotation(ctx, "user_id", "abc123")
logger.Infof(ctx, "processed %d items", 3) // user_id=abc123 is included
```

## Development

```bash
make unit-test
make lint
```
