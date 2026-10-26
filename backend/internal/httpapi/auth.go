package httpapi

import (
	"net/http"
	"time"

	"github.com/SV1Stail/online_school/internal/middleware"
	"github.com/SV1Stail/online_school/internal/user"
)

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type updateOwnProfileRequest struct {
	FullName     *string `json:"fullName"`
	SubjectLabel *string `json:"subjectLabel"`
}

func (a *API) login(writer http.ResponseWriter, request *http.Request) {
	var input loginRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	result, err := a.auth.Login(
		request.Context(),
		input.Login,
		input.Password,
		middleware.ClientIPFromContext(request.Context()),
	)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	http.SetCookie(writer, &http.Cookie{
		Name:     a.cookieName,
		Value:    result.Token,
		Path:     "/",
		Expires:  result.ExpiresAt,
		MaxAge:   int(time.Until(result.ExpiresAt).Seconds()),
		Secure:   a.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(writer, http.StatusOK, result.User)
}

func (a *API) logout(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie(a.cookieName); err == nil {
		if err := a.auth.Logout(request.Context(), cookie.Value); err != nil {
			a.handleError(writer, request, err)
			return
		}
	}
	a.clearSessionCookie(writer)
	writer.WriteHeader(http.StatusNoContent)
}

func (a *API) me(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, currentUser(request))
}

func (a *API) updateOwnProfile(writer http.ResponseWriter, request *http.Request) {
	var input updateOwnProfileRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	current := currentUser(request)
	if input.FullName == nil && input.SubjectLabel == nil {
		writeError(writer, http.StatusBadRequest, "invalid_input", "request contains invalid data", requestID(request))
		return
	}
	if input.SubjectLabel != nil && current.Role != user.RoleTeacher {
		writeError(writer, http.StatusForbidden, "forbidden", "operation is forbidden", requestID(request))
		return
	}
	updated, err := a.users.Update(
		request.Context(),
		current.ID,
		current.ID,
		user.UpdateInput{FullName: input.FullName, SubjectLabel: input.SubjectLabel},
	)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, updated)
}

func (a *API) changePassword(writer http.ResponseWriter, request *http.Request) {
	var input changePasswordRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	if err := a.users.ChangeOwnPassword(
		request.Context(),
		currentUser(request),
		input.CurrentPassword,
		input.NewPassword,
	); err != nil {
		a.handleError(writer, request, err)
		return
	}
	a.clearSessionCookie(writer)
	writer.WriteHeader(http.StatusNoContent)
}

func (a *API) clearSessionCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{
		Name:     a.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		Secure:   a.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
