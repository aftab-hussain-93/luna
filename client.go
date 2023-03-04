package main

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

type rlClient struct {
	client              client
	requests            map[int64]int
	intervalInSeconds   int
	requestsPerInterval int
}

type client interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (resp *http.Response, err error)
	Post(url string, contentType string, body io.Reader) (resp *http.Response, err error)
}

func NewSlidingWindowRLClient(cl client, intervalInSeconds, requestsPerInterval int) *rlClient {
	return &rlClient{
		client:              cl,
		requests:            make(map[int64]int),
		intervalInSeconds:   intervalInSeconds,
		requestsPerInterval: requestsPerInterval,
	}
}

func (c *rlClient) Get(link string) (resp *http.Response, err error) {
	l, e := url.Parse(link)
	if e != nil {
		return nil, e
	}
	req := &http.Request{
		Method: "GET",
		URL:    l,
	}
	return c.Do(req)
}

func (c *rlClient) Post(link string, contentType string, body io.Reader) (resp *http.Response, err error) {
	l, e := url.Parse(link)
	if e != nil {
		return nil, e
	}
	bdy, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		bdy = io.NopCloser(body)
	}
	req := &http.Request{
		Method: "POST",
		URL:    l,
		Body:   bdy,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	}
	return c.Do(req)
}

func (c *rlClient) Do(req *http.Request) (*http.Response, error) {
	// get epoch time
	timeNow := time.Now().Unix()
	// check the interval start time
	intervalStart := timeNow - int64(c.intervalInSeconds)
	totalRequestsSinceIntervalStart := 0
	for i := 0; i <= c.intervalInSeconds; i++ {
		if _, ok := c.requests[intervalStart+int64(i)]; !ok {
			c.requests[intervalStart+int64(i)] = 0
		}

		totalRequestsSinceIntervalStart += c.requests[intervalStart+int64(i)]
	}
	if totalRequestsSinceIntervalStart >= c.requestsPerInterval {
		return nil, errors.New("rate limit exceeded")
	}
	if _, ok := c.requests[timeNow]; !ok {
		c.requests[timeNow] = 0
	}
	c.requests[timeNow]++
	return c.client.Do(req)
}
