package model

import "errors"

// Sentinel errors shared by adapters and services. Controllers map these to
// HTTP status codes.
var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("already exists")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid request")
)
