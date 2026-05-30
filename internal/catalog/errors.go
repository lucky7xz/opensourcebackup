package catalog

import "errors"

var (
	ErrNotFound = errors.New("catalog: not found")
	ErrConflict = errors.New("catalog: conflict")
)
