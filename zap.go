package gotils

import (
	"context"
	"log"

	"github.com/rs/xid"
	"go.uber.org/zap"
)

/**********************
Note: all the zap related stuff is deprecated and will be removed at some point.
*****************/

// WithLogger call this when you first startup your app
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerContextKey, l)
}

// Logger get the logger
func Logger(ctx context.Context) *zap.Logger {
	var logger *zap.Logger
	loggerInterface := ctx.Value(LoggerContextKey)
	if loggerInterface != nil {
		if lgr, ok := loggerInterface.(*zap.Logger); ok {
			logger = lgr
		}
	}
	if logger == nil {
		// add a default one to context and return it
		logger, err := zap.NewProduction()
		if err != nil {
			log.Fatalf("can't initialize zap logger: %v", err)
		}
		return logger
	}
	return logger
}

// L shortcut for Logger
func L(ctx context.Context) *zap.Logger {
	return Logger(ctx)
}

// WithRequestID stores a request ID into the context
func WithRequestID(ctx context.Context) context.Context {
	guid := xid.New()
	gs := guid.String()
	ctx = context.WithValue(ctx, contextKey(RequestIDContextKey), gs)
	ctx = AddFields(ctx, zap.String("request_id", gs))
	return ctx
}

// AddFields adds fields to the context logger
func AddFields(ctx context.Context, fields ...zap.Field) context.Context {
	l := Logger(ctx)
	l = l.With(fields...)
	ctx = WithLogger(ctx, l)
	return ctx
}

/********************
END zap related stuff
*********************/
