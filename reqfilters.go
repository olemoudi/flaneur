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
	//	if r.URL.String() != newurl.String() {
	//		debug("normalize:", r.URL.String(), "to", newurl.String())
	//	}
	req, err := http.NewRequest("GET", newurl.String(), nil)
	if err != nil {
		return nil
	}
	return req
}

//var bfilter BloomFilter
//var bfilter *bloom.BloomFilter = bloom.New(80000000, 5)

func urlSeen(r *http.Request) *http.Request {
	if bfilter.TestAndAddString(r.URL.String()) {
		return nil
	}
	return r
	/*
		_, dup := seen[strings.TrimSpace(r.URL.String())]
		if dup {
			return nil
		}
		seen[strings.TrimSpace(r.URL.String())] = struct{}{}*/
}
