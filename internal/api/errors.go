package api

import (
	"net/http"
	"time"
)

type Error struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Status   int    `json:"status"`
	Upstream string `json:"upstream"`
}

func (e *Error) Error() string { return e.Message }

func (e *Error) ExitCode() int {
	switch e.Code {
	case "bad_request":
		return 2
	case "unauthorized", "missing_api_key":
		return 3
	case "not_found":
		return 4
	case "quota_exceeded":
		return 5
	default:
		return 1
	}
}

func responseError(status int, upstream, reset string) *Error {
	e := &Error{Status: status, Upstream: upstream}
	switch status {
	case http.StatusBadRequest:
		e.Code, e.Message = "bad_request", "the API rejected the request parameters"
	case http.StatusUnauthorized, http.StatusForbidden:
		e.Code, e.Message = "unauthorized", "the API key is invalid, expired, or not authorized for this endpoint"
	case http.StatusNotFound:
		e.Code, e.Message = "not_found", "the requested resource was not found"
	case http.StatusTooManyRequests:
		e.Code, e.Message = "quota_exceeded", "API quota or rate limit exceeded"
		if duration, err := time.ParseDuration(reset + "s"); err == nil && duration >= 0 {
			e.Message += ", resets in " + duration.String()
		}
	default:
		e.Code, e.Message = "upstream_error", "the API returned an unexpected HTTP status"
	}
	return e
}
