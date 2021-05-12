package client

import "errors"

var (
	ErrPermission     = errors.New("permission denied")
	ErrNotImplemented = errors.New("test not implemented for protocol")
)
