package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/SV1Stail/online_school/internal/schedule"
	"github.com/SV1Stail/online_school/internal/user"
)

func (a *API) createAvailability(writer http.ResponseWriter, request *http.Request) {
	var input schedule.CreateAvailabilityInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	if input.TeacherID != "" {
		if _, err := uuid.Parse(input.TeacherID); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
			return
		}
	}
	result, err := a.schedule.CreateAvailability(request.Context(), currentUser(request), input)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	status := http.StatusCreated
	if len(result.Items) == 0 {
		status = http.StatusOK
	}
	writeJSON(writer, status, result)
}

func (a *API) listAvailability(writer http.ResponseWriter, request *http.Request) {
	teacherID := request.PathValue("teacherID")
	if _, err := uuid.Parse(teacherID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
		return
	}
	from, to, ok := parsePeriod(request, 90*24*time.Hour)
	if !ok {
		writeError(writer, http.StatusBadRequest, "invalid_period", "invalid from or to value", requestID(request))
		return
	}
	items, err := a.schedule.ListAvailability(request.Context(), teacherID, from, to)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}

func (a *API) listAvailableSlots(writer http.ResponseWriter, request *http.Request) {
	teacherID := request.PathValue("teacherID")
	if _, err := uuid.Parse(teacherID); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid teacher id", requestID(request))
		return
	}
	from, to, ok := parsePeriod(request, 30*24*time.Hour)
	if !ok {
		writeError(writer, http.StatusBadRequest, "invalid_period", "invalid from or to value", requestID(request))
		return
	}
	items, err := a.schedule.ListAvailableSlots(request.Context(), currentUser(request), teacherID, from, to)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}

func (a *API) deleteAvailability(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("availabilityID")
	if _, err := uuid.Parse(id); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid availability id", requestID(request))
		return
	}
	if err := a.schedule.DeleteAvailability(request.Context(), currentUser(request), id); err != nil {
		a.handleError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (a *API) createBooking(writer http.ResponseWriter, request *http.Request) {
	var input schedule.CreateBookingInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_json", "invalid request body", requestID(request))
		return
	}
	for _, value := range []string{input.AvailabilityID, input.TeacherID, input.StudentID} {
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_id", "invalid booking reference", requestID(request))
			return
		}
	}
	input.IdempotencyKey = request.Header.Get("Idempotency-Key")
	created, err := a.schedule.CreateBooking(request.Context(), currentUser(request), input)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (a *API) listBookings(writer http.ResponseWriter, request *http.Request) {
	from, to, ok := parsePeriod(request, 90*24*time.Hour)
	if !ok {
		writeError(writer, http.StatusBadRequest, "invalid_period", "invalid from or to value", requestID(request))
		return
	}
	items, err := a.schedule.ListBookings(request.Context(), currentUser(request), from, to)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}

func (a *API) confirmBooking(writer http.ResponseWriter, request *http.Request) {
	a.changeBookingStatus(writer, request, a.schedule.Confirm)
}

func (a *API) rejectBooking(writer http.ResponseWriter, request *http.Request) {
	a.changeBookingStatus(writer, request, a.schedule.Reject)
}

func (a *API) cancelBooking(writer http.ResponseWriter, request *http.Request) {
	a.changeBookingStatus(writer, request, a.schedule.Cancel)
}

func (a *API) changeBookingStatus(
	writer http.ResponseWriter,
	request *http.Request,
	change func(context.Context, *user.User, string) (*schedule.Booking, error),
) {
	id, ok := a.bookingID(request)
	if !ok {
		writeError(writer, http.StatusBadRequest, "invalid_id", "invalid booking id", requestID(request))
		return
	}
	updated, err := change(request.Context(), currentUser(request), id)
	if err != nil {
		a.handleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, updated)
}

func (a *API) bookingID(request *http.Request) (string, bool) {
	id := request.PathValue("bookingID")
	if _, err := uuid.Parse(id); err != nil {
		return "", false
	}
	return id, true
}

func parsePeriod(request *http.Request, defaultSpan time.Duration) (time.Time, time.Time, bool) {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now.Add(defaultSpan)
	var err error
	if value := request.URL.Query().Get("from"); value != "" {
		from, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
	}
	if value := request.URL.Query().Get("to"); value != "" {
		to, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
	}
	return from.UTC(), to.UTC(), to.After(from)
}
