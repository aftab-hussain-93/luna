# Luna

Luna is a http client with build in functionality such as rate limiting, backoffs, retries. It's highly extensible allowing users to provide any data source for sliding window rate limiter fallback functions incase the request fails. It can be used to send http requests at scale. It's named after my cat :).

## Sliding Window Rate Limited HTTP Client

Instructions on how to use it

## Feature List

- Sliding Window Rate Limiter
- Examples (including Redis datastore)
- Retry mechanism at each request level
- Structured logging per request with multiple levels and output stream

## Features required
- Request coalescing
- Create wrapper request with req id, name
- Set up retry and sliding window rate limiter; Redis single/cluster deployment support
- Simpler get, post etc with res body, res error body, ok statuses
- OpenTelemetry support
