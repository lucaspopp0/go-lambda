package middleware

import (
	"context"
	"log"
	"slices"

	"github.com/lucaspopp0/go-lambda/eventwrappers"
)

type CORSOptions struct {
	ValidOrigins []string
}

var ValidOrigins = []string{
	"http://localhost:8081",
}

func WithCORS(opts CORSOptions) func(next HandlerFunc) HandlerFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req eventwrappers.HttpRequest) (eventwrappers.HttpResponse, error) {
			requestOrigin := req.Header("Origin")

			response, err := next(ctx, req)
			if err != nil {
				log.Printf("caught error: %v", err)

				response = eventwrappers.NewResponseFor(req)
				if response == nil {
					// Unsupported event type; nothing to build an error response from
					return nil, err
				}

				response.SetStatusCode(500)
				response.SetBody(err.Error())
			}

			response.SetHeader("Access-Control-Allow-Methods", "*")
			response.SetHeader("Access-Control-Allow-Headers", "*")
			response.SetHeader("Access-Control-Allow-Credentials", "true")

			if requestOrigin != "" && slices.Contains(opts.ValidOrigins, requestOrigin) {
				response.SetHeader("Access-Control-Allow-Origin", requestOrigin)
			}

			return response, nil
		}
	}
}
