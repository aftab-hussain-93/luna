package luna

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type slidingWindowRLClient struct {
	mu                  sync.Mutex
	client              client
	store               rlStorage
	intervalInSeconds   int
	requestsPerInterval int
	allowWait           bool
}

type client interface {
	Do(req *http.Request) (*http.Response, error)
}

type rlStorage interface {
	// get the number of requests made in the provided interval
	GetRequestsCountInInterval(ctx context.Context, start, end time.Time) (int, error)
	// increment the number of requests made in the current interval
	IncrementRequestCount(ctx context.Context, key time.Time) error
}
