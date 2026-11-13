package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/SV1Stail/online_school/internal/logging"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	count, err := w.ResponseWriter.Write(data)
	w.bytes += count
	return count, err
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := randomID()
		ctx := context.WithValue(request.Context(), requestIDContextKey, requestID)
		ctx = logging.WithRequestID(ctx, requestID)
		writer.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func AccessLog(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			wrapped := &statusWriter{ResponseWriter: writer}
			next.ServeHTTP(wrapped, request)
			logger.Info().
				Str("event", "http_request").
				Str("request_id", RequestIDFromContext(request.Context())).
				Str("method", request.Method).
				Str("path", request.URL.Path).
				Str("client_ip", ClientIPFromContext(request.Context())).
				Int("status", wrapped.status).
				Int("response_bytes", wrapped.bytes).
				Dur("duration", time.Since(started)).
				Msg("HTTP request completed")
		})
	}
}

func Recover(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error().
						Interface("panic", recovered).
						Str("request_id", RequestIDFromContext(request.Context())).
						Msg("recovered HTTP panic")
					http.Error(writer, `{"error":{"code":"internal_error","message":"internal server error"}}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		writer.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(writer, request)
	})
}

func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			request.Body = http.MaxBytesReader(writer, request.Body, maxBytes)
			next.ServeHTTP(writer, request)
		})
	}
}

func ResolveClientIP(trustedCIDRs []string) (func(http.Handler) http.Handler, error) {
	trusted, err := parseCIDRs(trustedCIDRs)
	if err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			clientIP := remoteIP(request.RemoteAddr)
			if isTrusted(clientIP, trusted) {
				clientIP = forwardedClientIP(request.Header.Get("X-Forwarded-For"), clientIP)
			}
			ctx := context.WithValue(request.Context(), clientIPContextKey, clientIP)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}, nil
}

func randomID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(buffer)
}

func parseCIDRs(values []string) ([]*net.IPNet, error) {
	result := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, err
		}
		result = append(result, network)
	}
	return result, nil
}

func isTrusted(value string, networks []*net.IPNet) bool {
	ip := net.ParseIP(value)
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteIP(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}
	return address
}

func forwardedClientIP(header, fallback string) string {
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return fallback
	}
	candidate := strings.TrimSpace(parts[0])
	if net.ParseIP(candidate) == nil {
		return fallback
	}
	return candidate
}
