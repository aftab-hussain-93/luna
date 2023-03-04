package main

import (
	"io"
	"net/http"
	"testing"
)

type mockClient struct {
	count int
}

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

func TestDo(t *testing.T) {
	var mockClient = &mockClient{}
	var rlClient = NewSlidingWindowRLClient(mockClient, 10, 50)
	for i := 0; i < 50; i++ {
		if _, err := rlClient.Do(&http.Request{}); err != nil {
			t.Errorf("Expected no error, got %s", err)
		}
	}
	if _, err := rlClient.Do(&http.Request{}); err == nil {
		t.Errorf("Expected error, got nil")
	}
	if mockClient.count != 50 {
		t.Errorf("Expected 50 requests, got %d", mockClient.count)
	}

}
