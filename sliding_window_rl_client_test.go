package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type mockClient struct {
	count int
}

type mockStorage map[int64]int

func (c *mockClient) Do(req *http.Request) (*http.Response, error) {
	c.count++
	return nil, nil
}
func (c *mockClient) Get(url string) (resp *http.Response, err error) {
	return nil, nil
}

func (c *mockClient) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	return nil, nil
}

func (s mockStorage) GetRequestsCountInInterval(ctx context.Context, start, end time.Time) (int, error) {
	endUnix := end.Unix()
	intervalStartInUnix := start.Unix()
	cnt := 0
	for i := intervalStartInUnix; i <= endUnix; i++ {
		if _, ok := s[i]; ok {
			cnt += s[i]
		}
	}
	return cnt, nil
}

func (s mockStorage) IncrementRequestCount(ctx context.Context, key time.Time) error {
	s[key.Unix()]++
	return nil
}

type doTest struct {
	intervalInSeconds       int
	requestsPerInterval     int
	requestedCount          int
	requestReceivedExpected int
}

var doTests = []doTest{
	{10, 50, 50, 50},
	{10, 50, 100, 50},
}

func TestDo(t *testing.T) {
	for _, tt := range doTests {
		mc := &mockClient{}
		mockStorage := mockStorage{}
		rlClient := NewSlidingWindowRLClient(mc, tt.intervalInSeconds, tt.requestsPerInterval, mockStorage, false)
		ctx := context.Background()
		for i := 0; i < tt.requestedCount; i++ {
			rlClient.Do(ctx, &http.Request{})
		}
		if mc.count != tt.requestReceivedExpected {
			t.Errorf("Expected %d requests, got %d", tt.requestReceivedExpected, mc.count)
		}
	}
}

func TestDoRateLimitShouldError(t *testing.T) {
	mc := &mockClient{}
	mockStorage := mockStorage{}
	shouldError := false
	rlClient := NewSlidingWindowRLClient(mc, 1, 1, mockStorage, shouldError)
	ctx := context.Background()
	_, err := rlClient.Do(ctx, &http.Request{})
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	_, err = rlClient.Do(ctx, &http.Request{})
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Expected error %s, got %s", ErrRateLimitExceeded, err)
	}
}

func TestDoRateLimitShouldWait(t *testing.T) {
	mc := &mockClient{}
	mockStorage := mockStorage{}
	allowWait := true
	rlClient := NewSlidingWindowRLClient(mc, 2, 1, mockStorage, allowWait)
	ctx := context.Background()
	_, err := rlClient.Do(ctx, &http.Request{})
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	_, err = rlClient.Do(ctx, &http.Request{})
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestCtxDeadline(t *testing.T) {
	mc := &mockClient{}
	mockStorage := mockStorage{}
	allowWait := true
	rlClient := NewSlidingWindowRLClient(mc, 1, 1, mockStorage, allowWait)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancel()
	_, err := rlClient.Do(ctx, &http.Request{})
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	_, err = rlClient.Do(ctx, &http.Request{})
	if !errors.Is(err, ErrTimeOut) {
		t.Errorf("Expected error %s, got %s", ErrTimeOut, err)
	}
}

func TestCtxCancel(t *testing.T) {
	mc := &mockClient{}
	mockStorage := mockStorage{}
	allowWait := true
	rlClient := NewSlidingWindowRLClient(mc, 1, 1, mockStorage, allowWait)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	cancel()
	_, err := rlClient.Do(ctx, &http.Request{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected error %s, got %s", context.Canceled, err)
	}
}
