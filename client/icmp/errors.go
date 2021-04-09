package icmp

import "errors"

var (
	ErrTTLExceeded = errors.New("packet ttl exceeded")
)
