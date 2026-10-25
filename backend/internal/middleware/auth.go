package middleware

import (
	"context"
	"net/http"

	"github.com/SV1Stail/online_school/internal/auth"
	"github.com/SV1Stail/online_school/internal/user"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*user.User, error)
}

func Authenticate(service Authenticator, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			cookie, err := request.Cookie(cookieName)
			if err != nil {
				writeUnauthorized(writer)
				return
			}
			currentUser, err := service.Authenticate(request.Context(), cookie.Value)
			if err != nil {
				writeUnauthorized(writer)
				return
			}
			ctx := context.WithValue(request.Context(), userContextKey, currentUser)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func RequireRoles(roles ...user.Role) func(http.Handler) http.Handler {
	allowed := make(map[user.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			currentUser, ok := UserFromContext(request.Context())
			if !ok {
				writeUnauthorized(writer)
				return
			}
			if _, ok := allowed[currentUser.Role]; !ok {
				http.Error(writer, `{"error":{"code":"forbidden","message":"forbidden"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func RequirePasswordChanged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		currentUser, ok := UserFromContext(request.Context())
		if !ok {
			writeUnauthorized(writer)
			return
		}
		if currentUser.MustChangePassword &&
			request.URL.Path != "/api/v1/auth/password" &&
			request.URL.Path != "/api/v1/me" {
			http.Error(
				writer,
				`{"error":{"code":"password_change_required","message":"temporary password must be changed"}}`,
				http.StatusForbidden,
			)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func writeUnauthorized(writer http.ResponseWriter) {
	http.Error(writer, `{"error":{"code":"unauthorized","message":"authentication required"}}`, http.StatusUnauthorized)
}

var _ Authenticator = (*auth.Service)(nil)
