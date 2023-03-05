package luna

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Creates a new sliding window rate limited client
// intervalInSeconds: the interval in seconds
// requestsPerInterval: the number of requests allowed in the interval
// st: the storage to use
// allowWait: if true, the client will wait until the next interval to send the request, if false, it will return an error
func NewSlidingWindowRLClient(cl client, intervalInSeconds, requestsPerInterval int, st rlStorage, allowWait bool) *slidingWindowRLClient {
	return &slidingWindowRLClient{
		client:              cl,
		intervalInSeconds:   intervalInSeconds,
		requestsPerInterval: requestsPerInterval,
		store:               st,
		allowWait:           allowWait,
	}
}

// Get sends a GET request to the provided link, it's a wrapper around rate limited Do method that ensures that the rate limit is not exceeded
// link: the link to send the request to
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

// Post sends a POST request to the provided link, it's a wrapper around rate limited Do method that ensures that the rate limit is not exceeded
// link: the link to send the request to
// contentType: the content type of the body
// body: the body of the request
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
	allowWait := false
	c.mu.Lock()
	allowWait = c.allowWait
	c.mu.Unlock()

	// check if rate limit is already exceeded
	hasExceeded, err := c.hasExceededRateLimit(ctx)
	if err != nil {
		return nil, err
	}
	// if rate limit is not exceeded, send the request
	if !hasExceeded {
		return c.sendRequest(ctx, req)
	}

	// rate limit is exceeded, check if we can wait
	if !allowWait {
		return nil, ErrRateLimitExceeded
	}

	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	return c.sendRequest(ctx, req)
}

func (c *slidingWindowRLClient) sendRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.store.IncrementRequestCount(ctx, time.Now()); err != nil {
		return nil, ErrAddingRequestCount
	}
	return c.client.Do(req)
}

// wait function waits until the next possible interval to send the request
func (c *slidingWindowRLClient) wait(ctx context.Context) error {
	nextOpenWindow, err := c.findNextOpenWindow(ctx)
	if err != nil {
		return err
	}
	// checking if the context has a deadline and if the next open window is after the deadline
	if deadline, ok := ctx.Deadline(); ok && nextOpenWindow.After(deadline) {
		return ErrTimeOut
	}
	t := time.NewTimer(time.Until(nextOpenWindow))
	// we wait until the next open window or until the context is done
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// hasExceededRateLimit checks if the rate limit is exceeded
func (c *slidingWindowRLClient) hasExceededRateLimit(ctx context.Context) (bool, error) {
	c.mu.Lock()
	rlIntervalSec := c.intervalInSeconds
	rlLimit := c.requestsPerInterval
	c.mu.Unlock()
	timeNow := time.Now()
	intervalStart := timeNow.Add(time.Duration(-rlIntervalSec) * time.Second)

	currentCount, err := c.store.GetRequestsCountInInterval(ctx, intervalStart, timeNow)
	if err != nil {
		return false, ErrGettingRequestsCount
	}
	return currentCount >= rlLimit, nil
}

func (c *slidingWindowRLClient) findNextOpenWindow(ctx context.Context) (time.Time, error) {
	c.mu.Lock()
	rlIntervalSec := c.intervalInSeconds
	rlLimit := c.requestsPerInterval
	c.mu.Unlock()

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
