package pgcontext

import (
	"context"

	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"
)

type requestIDKey struct{}

// WithReuqestID appends request id to the context.
func WithRequestID(ctx context.Context, rid string) (out context.Context) {
	return context.WithValue(ctx, requestIDKey{}, rid)
}

// RequestID returns request id from context.
func RequestID(ctx context.Context) (requestID string, ok bool) {
	requestID, ok = ctx.Value(requestIDKey{}).(string)
	if ok {
		return requestID, true
	}

	return "", false
}

// GetOrMakeRequestID check if request id is empty or returned ok is false and
// created new UUID values.
func GetOrMakeRequestID(requestID string, ok bool) string {
	if !ok || len(requestID) == 0 {
		requestID = uuid.NewV4().String()
	}

	return requestID
}

func WithZerolog(ctx context.Context, zl zerolog.Logger) (out context.Context) {
	return zl.WithContext(ctx)
}

func Zerolog(ctx context.Context) (out *zerolog.Logger) {
	return zerolog.Ctx(ctx)
}
