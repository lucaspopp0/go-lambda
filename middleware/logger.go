package middleware

import (
	"context"
	"github.com/lucaspopp0/go-lambda/logs"
	"os"
)

type contextKey string

const (
	ctxKeyLogger contextKey = "middleware_logger"
)

func WithLogger[Req Request, Resp Response](next HandlerFunc[Req, Resp]) HandlerFunc[Req, Resp] {
	return func(ctx context.Context, event Req) (Resp, error) {
		logger := logs.NewLogger(logs.LoggerOpts{
			Out: os.Stdout,
		})

		ctx = context.WithValue(ctx, ctxKeyLogger, logger)

		ctx = logs.WithAnnotation(ctx, "request", event)

		logger.Info(ctx, "Request received")

		response, err := next(ctx, event)
		if err != nil {
			return response, err
		}

		ctx = logs.WithAnnotation(ctx, "response", response)

		logResponse := logger.Info
		if statusCode(response) >= 500 {
			logResponse = logger.Error
		} else if statusCode(response) >= 400 {
			logResponse = logger.Warn
		}

		logResponse(ctx, "Sending response")

		return response, nil
	}
}

func GetLogger(ctx context.Context) logs.Logger {
	if logger, ok := ctx.Value(ctxKeyLogger).(logs.Logger); ok {
		return logger
	}

	logger := logs.NewLogger(logs.LoggerOpts{
		Out: os.Stdout,
	})

	logger.Warn(ctx, "No logger found in context. Falling back to a basic default logger")

	return logger
}
