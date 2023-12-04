package types

import "errors"

var (
	ErrUnknownType error = errors.New("unknown metric type")
)
