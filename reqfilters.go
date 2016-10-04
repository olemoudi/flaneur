package main

import (
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/purell"
)

func dummyFilter(req *http.Request) *http.Request {
	return req
}

func normalizeURL(r *http.Request) *http.Request {
	normalized := purell.NormalizeURL(r.URL, purell.FlagsSafe|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagSortQuery)
	newurl, err := url.Parse(normalized)
	if err != nil {
		debug("Error parsing normalized URL", normalized)
		newurl = nil
	}
	req, err := http.NewRequest("GET", newurl.String(), nil)
	if err != nil {
		return nil
	}
	return req
}
