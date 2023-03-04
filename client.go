package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type slidingWindowRLClient struct {
	mu                  sync.Mutex
	client              client
	store               storage
	intervalInSeconds   int
	requestsPerInterval int
}

type client interface {
	Do(req *http.Request) (*http.Response, error)
}

type storage interface {
	// get the number of requests made in the provided interval
	GetRequestsCountInInterval(ctx context.Context, start, end time.Time) (int, error)
	// increment the number of requests made in the current interval
	IncrementRequestCount(ctx context.Context, key time.Time) error
}

func NewSlidingWindowRLClient(cl client, intervalInSeconds, requestsPerInterval int, st storage) *slidingWindowRLClient {
	return &slidingWindowRLClient{
		client:              cl,
		intervalInSeconds:   intervalInSeconds,
		requestsPerInterval: requestsPerInterval,
		store:               st,
	}
}

func (c *slidingWindowRLClient) Get(ctx context.Context, link string) (resp *http.Response, err error) {
	l, e := url.Parse(link)
	if e != nil {
		return nil, e
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    l,
	}
	return c.Do(ctx, req)
}

func (c *slidingWindowRLClient) Post(ctx context.Context, link string, contentType string, body io.Reader) (resp *http.Response, err error) {
	l, e := url.Parse(link)
	if e != nil {
		return nil, e
	}
	bdy, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		bdy = io.NopCloser(body)
	}
	req := &http.Request{
		Method: http.MethodPost,
		URL:    l,
		Body:   bdy,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	}
	return c.Do(ctx, req)
}

func (c *slidingWindowRLClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// check if context is done
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	c.mu.Lock()
	rlIntervalSec := c.intervalInSeconds
	rlLimit := c.requestsPerInterval
	c.mu.Unlock()

	// get epoch time
	timeNow := time.Now().Unix()
	intervalStart := timeNow - int64(rlIntervalSec)
	currCnt, err := c.store.GetRequestsCountInInterval(ctx, time.Unix(intervalStart, 0), time.Now())
	if err != nil {
		return nil, ErrGettingRequestsCount
	}
	if currCnt >= rlLimit {
		return nil, ErrRateLimitExceeded
	}
	if err := c.store.IncrementRequestCount(ctx, time.Now()); err != nil {
		return nil, ErrAddingRequestCount
	}
	return c.client.Do(req)
}
