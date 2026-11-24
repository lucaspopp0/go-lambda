package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/lucaspopp0/go-lambda/eventwrappers"
)

// HandlerFunc is a Lambda HTTP handler that works on the event-agnostic
// eventwrappers.HttpRequest / eventwrappers.HttpResponse views, so the same
// middleware works for API Gateway v2 and Lambda Function URL events.
//
// Use Adapt to turn a HandlerFunc into a function that can be passed to
// lambda.Start.
type HandlerFunc func(ctx context.Context, req eventwrappers.HttpRequest) (eventwrappers.HttpResponse, error)

// Adapt converts a HandlerFunc into a handler for the concrete Lambda event
// types Req and Res, e.g.
//
//	lambda.Start(middleware.Adapt[events.APIGatewayV2HTTPRequest, events.APIGatewayV2HTTPResponse](handler))
func Adapt[Req eventwrappers.HttpRequestEvent, Res eventwrappers.HttpResponseEvent](h HandlerFunc) func(context.Context, Req) (Res, error) {
	return func(ctx context.Context, event Req) (Res, error) {
		var zero Res

		resp, err := h(ctx, eventwrappers.WrapRequest(event))
		if err != nil {
			return zero, err
		}
		if resp == nil {
			return zero, errors.New("middleware: handler returned a nil response")
		}

		out, ok := eventwrappers.UnwrapResponse[Res](resp)
		if !ok {
			return zero, fmt.Errorf("middleware: handler returned a response of type %T, which does not match the expected %T", resp.Unwrap(), zero)
		}

		return out, nil
	}
}
