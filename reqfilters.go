package main

import "net/http"

func dummyFilter(req *http.Request) *http.Request {
	return req
}
