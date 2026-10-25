package httpapi

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/SV1Stail/online_school/internal/conference"
)

func (a *API) listTeacherConferenceLinks(writer http.ResponseWriter, request *http.Request) {
	teacherID := request.PathValue("teacherID")
	if _, err := uuid.Parse(teacherID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
		return
	}
	items, err := a.conference.ListForTeacher(request.Context(), currentUser(request), teacherID)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}

func (a *API) listBookingConferenceLinks(writer http.ResponseWriter, request *http.Request) {
	bookingID := request.PathValue("bookingID")
	if _, err := uuid.Parse(bookingID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid booking id", requestID(request))
		return
	}
	items, err := a.conference.ListForBooking(request.Context(), currentUser(request), bookingID)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createConferenceLink(writer http.ResponseWriter, request *http.Request) {
	teacherID := request.PathValue("teacherID")
	if _, err := uuid.Parse(teacherID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
		return
	}
	var input conference.CreateInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	input.TeacherID = teacherID
	created, err := a.conference.Create(request.Context(), currentUser(request), input)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (a *API) updateConferenceLink(writer http.ResponseWriter, request *http.Request) {
	teacherID := request.PathValue("teacherID")
	linkID := request.PathValue("linkID")
	if _, err := uuid.Parse(teacherID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
		return
	}
	if _, err := uuid.Parse(linkID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid conference link id", requestID(request))
		return
	}
	var input conference.UpdateInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	updated, err := a.conference.Update(request.Context(), currentUser(request), linkID, teacherID, input)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, updated)
}

func (a *API) deleteConferenceLink(writer http.ResponseWriter, request *http.Request) {
	teacherID := request.PathValue("teacherID")
	linkID := request.PathValue("linkID")
	if _, err := uuid.Parse(teacherID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
		return
	}
	if _, err := uuid.Parse(linkID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid conference link id", requestID(request))
		return
	}
	if err := a.conference.Delete(request.Context(), currentUser(request), linkID, teacherID); err != nil {
		a.handleError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
