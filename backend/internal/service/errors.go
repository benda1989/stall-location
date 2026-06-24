package service

import (
	"net/http"

	"gkk/expect"
)

func badRequest(code int, message string) error {
	return expect.New(http.StatusBadRequest, code, message)
}

func forbidden(message string) error {
	return expect.New(http.StatusForbidden, expect.CodeAuthForbidden, message)
}

func notFound(message string) error {
	return expect.New(http.StatusNotFound, expect.CodeCommonNotFound, message)
}

func conflict(code int, message string) error {
	return expect.New(http.StatusConflict, code, message)
}

func validation(message string) error {
	return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, message)
}
