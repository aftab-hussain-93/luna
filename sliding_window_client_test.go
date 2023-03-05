package luna

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type mockSwRlClient struct {
	count int
}

type mockSwRlStorage map[int64]int

func (c *mockSwRlClient) Do(req *http.Request) (*http.Response, error) {
	c.count++
	return nil, nil
}
func (c *mockSwRlClient) Get(url string) (resp *http.Response, err error) {
	return nil, nil
}

func (c *mockSwRlClient) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	return nil, nil
}

func (s mockSwRlStorage) GetRequestsCountInInterval(ctx context.Context, start, end time.Time) (int, error) {
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

func (s mockSwRlStorage) IncrementRequestCount(ctx context.Context, key time.Time) error {
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
	{10, 40, 100, 40},
}

func TestDo(t *testing.T) {
	for _, tt := range doTests {
		mc := &mockSwRlClient{}
		mockStorage := mockSwRlStorage{}
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
	mc := &mockSwRlClient{}
	mockStorage := mockSwRlStorage{}
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
	mc := &mockSwRlClient{}
	mockStorage := mockSwRlStorage{}
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
	mc := &mockSwRlClient{}
	mockStorage := mockSwRlStorage{}
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
	mc := &mockSwRlClient{}
	mockStorage := mockSwRlStorage{}
	allowWait := true
	rlClient := NewSlidingWindowRLClient(mc, 1, 1, mockStorage, allowWait)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	cancel()
	_, err := rlClient.Do(ctx, &http.Request{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected error %s, got %s", context.Canceled, err)
	}
}
