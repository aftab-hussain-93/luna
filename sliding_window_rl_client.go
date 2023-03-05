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
	allowWait           bool
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

func NewSlidingWindowRLClient(cl client, intervalInSeconds, requestsPerInterval int, st storage, allowWait bool) *slidingWindowRLClient {
	return &slidingWindowRLClient{
		client:              cl,
		intervalInSeconds:   intervalInSeconds,
		requestsPerInterval: requestsPerInterval,
		store:               st,
		allowWait:           allowWait,
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
	// check if context is already done
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	hasExceeded, err := c.hasExceededRateLimit(ctx)
	if err != nil {
		return nil, err
	}
	if !hasExceeded {
		return c.sendRequest(ctx, req)
	}
	// rate limit exceeded, return error or wait
	if !c.allowWait {
		return nil, ErrRateLimitExceeded
	}
	return c.waitDoer(ctx, req)
}

func (c *slidingWindowRLClient) sendRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.store.IncrementRequestCount(ctx, time.Now()); err != nil {
		return nil, ErrAddingRequestCount
	}
	return c.client.Do(req)
}

func (c *slidingWindowRLClient) waitDoer(ctx context.Context, req *http.Request) (*http.Response, error) {
	// find next open window
	nextOpenWindow, err := c.findNextOpenWindow(ctx)
	if err != nil {
		return nil, err
	}
	dl, hasDl := ctx.Deadline()
	if hasDl && nextOpenWindow.After(dl) {
		return nil, ErrTimeOut
	}
	t := time.NewTimer(time.Until(nextOpenWindow))
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.C:
		return c.sendRequest(ctx, req)
	}
}

func (c *slidingWindowRLClient) hasExceededRateLimit(ctx context.Context) (bool, error) {
	c.mu.Lock()
	rlIntervalSec := c.intervalInSeconds
	rlLimit := c.requestsPerInterval
	c.mu.Unlock()
	// get epoch time
	timeNow := time.Now()
	intervalStart := timeNow.Add(time.Duration(-rlIntervalSec) * time.Second)
	currCnt, err := c.store.GetRequestsCountInInterval(ctx, intervalStart, timeNow)
	if err != nil {
		return false, ErrGettingRequestsCount
	}
	return currCnt >= rlLimit, nil
}

func (c *slidingWindowRLClient) findNextOpenWindow(ctx context.Context) (time.Time, error) {
	c.mu.Lock()
	rlIntervalSec := c.intervalInSeconds
	rlLimit := c.requestsPerInterval
	c.mu.Unlock()
	// get epoch time
	windowEnd := time.Now()
	windowStart := windowEnd.Add(time.Duration(-rlIntervalSec) * time.Second)
	for {
		currCnt, err := c.store.GetRequestsCountInInterval(ctx, windowStart, windowEnd)
		if err != nil {
			return time.Time{}, ErrGettingRequestsCount
		}
		if currCnt < rlLimit {
			return windowEnd, nil
		}
		// move window forward by 1 second
		windowStart = windowStart.Add(time.Second)
		windowEnd = windowEnd.Add(time.Second)
	}
}
