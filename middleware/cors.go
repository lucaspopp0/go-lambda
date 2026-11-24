package middleware

import (
	"context"
	"log"
	"slices"
)

type CORSOptions struct {
	ValidOrigins []string
}

var ValidOrigins = []string{
	"http://localhost:8081",
}

func WithCORS[Req Request, Resp Response](opts CORSOptions) func(next HandlerFunc[Req, Resp]) HandlerFunc[Req, Resp] {
	return func(next HandlerFunc[Req, Resp]) HandlerFunc[Req, Resp] {
		return func(ctx context.Context, event Req) (Resp, error) {
			requestOrigin := getOrigin(requestHeaders(event))

			response, err := next(ctx, event)
			if err != nil {
				log.Printf("caught error: %v", err)
				return errorResponse[Resp](500, err.Error()), nil
			}

			setHeader(&response, "Access-Control-Allow-Methods", "*")
			setHeader(&response, "Access-Control-Allow-Headers", "*")
			setHeader(&response, "Access-Control-Allow-Credentials", "true")

			if requestOrigin != "" && slices.Contains(opts.ValidOrigins, requestOrigin) {
				setHeader(&response, "Access-Control-Allow-Origin", requestOrigin)
			}

			return response, nil
		}
	}
}

func getOrigin(headers map[string]string) string {
	if origin, ok := headers["Origin"]; ok {
		return origin
	}
	if origin, ok := headers["origin"]; ok {
		return origin
	}
	return ""
}
