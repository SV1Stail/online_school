package middleware

import (
	"net/http"
	"net/url"
)

func SameOrigin(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if parsed, err := url.Parse(origin); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			allowed[parsed.Scheme+"://"+parsed.Host] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Method == http.MethodGet || request.Method == http.MethodHead || request.Method == http.MethodOptions {
				next.ServeHTTP(writer, request)
				return
			}
			origin := request.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(writer, request)
				return
			}
			if _, ok := allowed[origin]; !ok {
				http.Error(writer, `{"error":{"code":"invalid_origin","message":"request origin is not allowed"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}
