package main

import "errors"

var (
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrGettingRequestsCount = errors.New("datastore_error: error getting requests count")
	ErrAddingRequestCount   = errors.New("datastore_error: error adding request count")
	ErrTimeOut              = errors.New("timeout")
)
