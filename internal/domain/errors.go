package domain

import "errors"

var (
    ErrNotFound      = errors.New("resource not found")
    ErrAlreadyExists = errors.New("resource already exists")
    ErrValidation    = errors.New("validation error")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
)

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e ValidationError) Error() string {
    return e.Field + ": " + e.Message
}