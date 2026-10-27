package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/SV1Stail/online_school/internal/user"
)

func TestRequirePasswordChangedAllowsOnlyProfileReadAndPasswordChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		method             string
		path               string
		mustChangePassword bool
		wantStatus         int
	}{
		{name: "read profile", method: http.MethodGet, path: "/api/v1/me", mustChangePassword: true, wantStatus: http.StatusNoContent},
		{name: "change password", method: http.MethodPost, path: "/api/v1/auth/password", mustChangePassword: true, wantStatus: http.StatusNoContent},
		{name: "update profile", method: http.MethodPatch, path: "/api/v1/me", mustChangePassword: true, wantStatus: http.StatusForbidden},
		{name: "other protected route", method: http.MethodGet, path: "/api/v1/teachers", mustChangePassword: true, wantStatus: http.StatusForbidden},
		{name: "password already changed", method: http.MethodPatch, path: "/api/v1/me", mustChangePassword: false, wantStatus: http.StatusNoContent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(test.method, test.path, nil)
			request = request.WithContext(context.WithValue(
				request.Context(),
				userContextKey,
				&user.User{MustChangePassword: test.mustChangePassword},
			))
			response := httptest.NewRecorder()
			handler := RequirePasswordChanged(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusNoContent)
			}))

			handler.ServeHTTP(response, request)

			require.Equal(t, test.wantStatus, response.Code)
		})
	}
}
