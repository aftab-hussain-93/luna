# luna
Luna is a http client with build in functionality such as rate limiting, backoffs, retries. It's highly extensible allowing users to provide any data source for sliding window rate limiter fallback functions incase the request fails. It conforms to the standard http client interface and can be used to send http requests at scale. It's named after my cat :)
