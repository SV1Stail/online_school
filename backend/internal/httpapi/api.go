package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/SV1Stail/online_school/internal/auth"
	"github.com/SV1Stail/online_school/internal/conference"
	"github.com/SV1Stail/online_school/internal/config"
	"github.com/SV1Stail/online_school/internal/middleware"
	"github.com/SV1Stail/online_school/internal/schedule"
	"github.com/SV1Stail/online_school/internal/user"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type API struct {
	auth         *auth.Service
	users        *user.Service
	schedule     *schedule.Service
	conference   *conference.Service
	database     Pinger
	cache        Pinger
	logger       zerolog.Logger
	cookieName   string
	cookieSecure bool
}

func New(
	cfg *config.Config,
	authService *auth.Service,
	userService *user.Service,
	scheduleService *schedule.Service,
	conferenceService *conference.Service,
	database Pinger,
	cache Pinger,
	logger zerolog.Logger,
) (*API, error) {
	cookieName := "online_school_session"
	if cfg.CookieSecure {
		cookieName = "__Host-online_school_session"
	}
	return &API{
		auth:         authService,
		users:        userService,
		schedule:     scheduleService,
		conference:   conferenceService,
		database:     database,
		cache:        cache,
		logger:       logger,
		cookieName:   cookieName,
		cookieSecure: cfg.CookieSecure,
	}, nil
}

func (a *API) Handler(cfg *config.Config) (http.Handler, error) {
	root := http.NewServeMux()
	root.HandleFunc("GET /health/live", a.live)
	root.HandleFunc("GET /health/ready", a.ready)
	root.Handle("POST /api/v1/auth/login", middleware.ConcurrencyLimit(cfg.MaxLoginInFlight)(http.HandlerFunc(a.login)))
	root.HandleFunc("POST /api/v1/auth/logout", a.logout)

	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/auth/password", a.changePassword)
	protected.HandleFunc("GET /api/v1/me", a.me)
	protected.HandleFunc("PATCH /api/v1/me", a.updateOwnProfile)
	protected.HandleFunc("GET /api/v1/teachers", a.listTeachers)
	protected.HandleFunc("GET /api/v1/teachers/{teacherID}/availability", a.listAvailability)
	protected.HandleFunc("GET /api/v1/teachers/{teacherID}/slots", a.listAvailableSlots)
	protected.HandleFunc("GET /api/v1/bookings", a.listBookings)
	protected.HandleFunc("POST /api/v1/bookings", a.createBooking)
	protected.HandleFunc("POST /api/v1/bookings/{bookingID}/cancel", a.cancelBooking)
	protected.HandleFunc("GET /api/v1/bookings/{bookingID}/conference-links", a.listBookingConferenceLinks)
	protected.HandleFunc("GET /api/v1/students/{studentID}/conference-links", a.listStudentConferenceLinks)

	staffOnly := middleware.RequireRoles(user.RoleTeacher, user.RoleModerator)
	protected.Handle("GET /api/v1/students", staffOnly(http.HandlerFunc(a.listStudents)))
	protected.Handle("POST /api/v1/availability", staffOnly(http.HandlerFunc(a.createAvailability)))
	protected.Handle("DELETE /api/v1/availability/{availabilityID}", staffOnly(http.HandlerFunc(a.deleteAvailability)))
	protected.Handle("POST /api/v1/bookings/{bookingID}/confirm", staffOnly(http.HandlerFunc(a.confirmBooking)))
	protected.Handle("POST /api/v1/bookings/{bookingID}/reject", staffOnly(http.HandlerFunc(a.rejectBooking)))
	protected.Handle("POST /api/v1/students/{studentID}/conference-links", staffOnly(http.HandlerFunc(a.createConferenceLink)))
	protected.Handle("PATCH /api/v1/students/{studentID}/conference-links/{linkID}", staffOnly(http.HandlerFunc(a.updateConferenceLink)))
	protected.Handle("DELETE /api/v1/students/{studentID}/conference-links/{linkID}", staffOnly(http.HandlerFunc(a.deleteConferenceLink)))

	admin := http.NewServeMux()
	admin.HandleFunc("GET /api/v1/admin/users", a.listUsers)
	admin.HandleFunc("POST /api/v1/admin/users", a.createUser)
	admin.HandleFunc("PATCH /api/v1/admin/users/{userID}", a.updateUser)
	admin.HandleFunc("POST /api/v1/admin/users/{userID}/reset-password", a.resetPassword)
	protected.Handle("/api/v1/admin/", middleware.RequireRoles(user.RoleModerator)(admin))

	authenticated := middleware.Authenticate(a.auth, a.cookieName)(middleware.RequirePasswordChanged(protected))
	root.Handle("/api/v1/", authenticated)

	clientIP, err := middleware.ResolveClientIP(cfg.TrustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, cfg.MaxRateClients)
	var handler http.Handler = root
	handler = middleware.Timeout(cfg.RequestTimeout)(handler)
	handler = middleware.ConcurrencyLimit(cfg.MaxInFlight)(handler)
	handler = rateLimiter.Middleware(handler)
	handler = middleware.BodyLimit(cfg.MaxBodyBytes)(handler)
	handler = middleware.SameOrigin(cfg.AllowedOrigins)(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.AccessLog(a.logger)(handler)
	handler = middleware.Recover(a.logger)(handler)
	handler = clientIP(handler)
	handler = middleware.RequestID(handler)
	return handler, nil
}

func (a *API) live(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) ready(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if err := a.database.Ping(ctx); err != nil {
		writeError(writer, http.StatusServiceUnavailable, "database_unavailable", "service is not ready", requestID(request))
		return
	}
	if err := a.cache.Ping(ctx); err != nil {
		writeError(writer, http.StatusServiceUnavailable, "cache_unavailable", "service is not ready", requestID(request))
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func requestID(request *http.Request) string {
	return middleware.RequestIDFromContext(request.Context())
}

func currentUser(request *http.Request) *user.User {
	result, _ := middleware.UserFromContext(request.Context())
	return result
}

func (a *API) handleError(writer http.ResponseWriter, request *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "internal server error"

	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		status, code, message = http.StatusUnauthorized, "invalid_credentials", "invalid login or password"
	case errors.Is(err, auth.ErrTooManyAttempts):
		status, code, message = http.StatusTooManyRequests, "login_rate_limited", "too many login attempts"
	case errors.Is(err, auth.ErrUnauthorized):
		status, code, message = http.StatusUnauthorized, "unauthorized", "authentication required"
	case errors.Is(err, user.ErrNotFound), errors.Is(err, schedule.ErrNotFound), errors.Is(err, conference.ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "resource not found"
	case errors.Is(err, user.ErrLoginConflict), errors.Is(err, schedule.ErrConflict), errors.Is(err, conference.ErrConflict):
		status, code, message = http.StatusConflict, "conflict", "resource conflicts with existing data"
	case errors.Is(err, schedule.ErrForbidden), errors.Is(err, conference.ErrForbidden):
		status, code, message = http.StatusForbidden, "forbidden", "operation is forbidden"
	case errors.Is(err, schedule.ErrBookingClosed):
		status, code, message = http.StatusUnprocessableEntity, "booking_closed", "booking closes 12 hours before the lesson"
	case errors.Is(err, schedule.ErrCancellationClosed):
		status, code, message = http.StatusUnprocessableEntity, "cancellation_closed", "confirmed lessons can only be cancelled at least 24 hours in advance"
	case errors.Is(err, schedule.ErrInvalidTransition):
		status, code, message = http.StatusConflict, "invalid_status_transition", "booking status has already changed"
	case errors.Is(err, user.ErrInvalidInput), errors.Is(err, schedule.ErrInvalidInput), errors.Is(err, conference.ErrInvalidInput):
		status, code, message = http.StatusBadRequest, "invalid_input", "request contains invalid data"
	}
	if status >= http.StatusInternalServerError {
		a.logger.Error().Err(err).Str("request_id", requestID(request)).Msg("request failed")
	}
	writeError(writer, status, code, message, requestID(request))
}
