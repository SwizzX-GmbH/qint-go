package qint

import (
	"encoding/json"
	"fmt"
)

// QintError is returned for any non-2xx API response. It carries the HTTP
// status code together with the RFC 7807 problem-details fields returned by
// the API. Use errors.As to extract it:
//
//	var qerr *qint.QintError
//	if errors.As(err, &qerr) && qerr.StatusCode == 403 {
//		// missing scope, etc.
//	}
type QintError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Type is the problem-details "type" URI, if present.
	Type string
	// Title is the short, human-readable summary of the problem type.
	Title string
	// Detail is the human-readable explanation specific to this occurrence.
	Detail string
	// Body is the raw response body, retained for debugging.
	Body string
}

// problemDetails mirrors the RFC 7807 application/problem+json body.
type problemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func newQintError(status int, body []byte) *QintError {
	e := &QintError{StatusCode: status, Body: string(body)}
	var p problemDetails
	if err := json.Unmarshal(body, &p); err == nil {
		e.Type = p.Type
		e.Title = p.Title
		e.Detail = p.Detail
	}
	return e
}

// Error implements the error interface.
func (e *QintError) Error() string {
	switch {
	case e.Detail != "":
		return fmt.Sprintf("qint: HTTP %d: %s", e.StatusCode, e.Detail)
	case e.Title != "":
		return fmt.Sprintf("qint: HTTP %d: %s", e.StatusCode, e.Title)
	default:
		return fmt.Sprintf("qint: unexpected HTTP status %d", e.StatusCode)
	}
}
