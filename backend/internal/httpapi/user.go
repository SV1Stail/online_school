package httpapi

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/SV1Stail/online_school/internal/user"
)

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (a *API) listTeachers(writer http.ResponseWriter, request *http.Request) {
	teachers, err := a.users.ListTeachers(request.Context())
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": teachers})
}

func (a *API) listUsers(writer http.ResponseWriter, request *http.Request) {
	users, err := a.users.ListAll(request.Context())
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": users})
}

func (a *API) listStudents(writer http.ResponseWriter, request *http.Request) {
	students, err := a.users.ListStudents(request.Context())
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": students})
}

func (a *API) createUser(writer http.ResponseWriter, request *http.Request) {
	var input user.CreateInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	created, err := a.users.Create(request.Context(), currentUser(request).ID, input)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (a *API) updateUser(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("userID")
	if _, err := uuid.Parse(id); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid user id", requestID(request))
		return
	}
	var input user.UpdateInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	updated, err := a.users.Update(request.Context(), currentUser(request).ID, id, input)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, updated)
}

func (a *API) resetPassword(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("userID")
	if _, err := uuid.Parse(id); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid user id", requestID(request))
		return
	}
	var input resetPasswordRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	if err := a.users.ResetPassword(request.Context(), currentUser(request).ID, id, input.Password); err != nil {
		a.handleError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
