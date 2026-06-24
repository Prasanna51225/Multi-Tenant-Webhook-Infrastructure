package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/webhook-platform/internal/domain"
)

type ErrorResponse struct {
	Error   string                   `json:"error"`
	Code    string                   `json:"code"`
	Details []domain.ValidationError `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{
		Error: message,
		Code:  http.StatusText(status),
	})
}

func MapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, domain.ErrAlreadyExists):
		return http.StatusConflict, "resource already exists"
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func HandleError(w http.ResponseWriter, err error) {
	status, message := MapError(err)
	Error(w, status, message)
}
