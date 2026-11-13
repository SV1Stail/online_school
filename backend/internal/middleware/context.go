package middleware

import (
	"context"

	"github.com/SV1Stail/online_school/internal/user"
)

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	clientIPContextKey  contextKey = "client_ip"
	userContextKey      contextKey = "user"
)

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey).(string)
	return value
}

func ClientIPFromContext(ctx context.Context) string {
	value, _ := ctx.Value(clientIPContextKey).(string)
	return value
}

func UserFromContext(ctx context.Context) (*user.User, bool) {
	value, ok := ctx.Value(userContextKey).(*user.User)
	return value, ok
}
