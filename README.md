# template
Template for new repositories

## humaserverless

Serve a [huma](https://huma.rocks) API from Lambda. Supports API Gateway v2 (HTTP API) and Lambda Function URL events.

```go
api := humaserverless.NewServerless(humaserverless.ApiOpts{Name: "my-api"})
// huma.Register(api, ...)

// API Gateway v2
lambda.Start(func(ctx context.Context, e events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return humaserverless.HttpHandler(ctx, api, e)
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
