package main

import "net/http"
import "github.com/PuerkitoBio/purell"

func dummyFilter(req *http.Request) *http.Request {
	return req
}

func normalizeURL(req *http.Request) *http.Request {
	normalized := purell.NormalizeURL(req.URL, purell.FlagsSafe | purell.FlagRemoveDotSegments | purell.FlagRemoveDuplicateSlashes | purell.FlagSortQuery)
	newurl, err := url.Parse(normalized)
	if err != nil {
		debug("Error parsing normalized URL", normalized)
		newurl = nil
	}
	return newurl
}
