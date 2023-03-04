package main

import "errors"

var (
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrGettingRequestsCount = errors.New("error getting requests count")
	ErrAddingRequestCount   = errors.New("error adding request count")
)
