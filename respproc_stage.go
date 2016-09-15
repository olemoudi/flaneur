package main

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const extractorCount = 10

func responseProcessor(id int) {

	for i := 1; i <= extractorCount; i++ {
		wg.Add(1)
		go extractLinks()
	}
	loop:
		for {
			select {
			case <-exiting:
				debug("processor exiting")
				break loop
			case <-time.After(time.Second * 5):
				continue loop// TODO: here goes the fan out

			}
			ping()

		}
	wg.Done()
	return
}

func extractLinks() {
	loop:
		for  {
			select {
			case <-exiting:
				debug("link extractor exiting")
				break loop
			case resp := <-downloadOutputQ:
				ping()
				extractLinksF(resp)
			}
		}
	wg.Done()
	return

}

func extractLinksF(resp *http.Response) {
	debug("extracting links")
	req, err := http.NewRequest("GET", "http://localhost:8000", nil)
	if err == nil {
		select {
		case reqFilterInputQ <- req:
		default:
			//case <-time.After(time.Millisecond * 0.5):
			//debug("link lost")
		}
}
	/*for i := 1; i <= 1; i++ {
		*output <- resp.Request.URL.String()
	}*/

	z := html.NewTokenizer(resp.Body)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			// Extract the href value, if there is one
			ok, link := getHref(t)
			if !ok {
				continue
			}

			// Make sure the url begines in http**
			hasProto := strings.Index(link, "http") == 0
			if hasProto {
				req, err := http.NewRequest("GET", link, nil)
				if err == nil {
					select {
					case reqFilterInputQ <- req:
					default:
						//case <-time.After(time.Millisecond * 0.5):
						//debug("link lost")
					}
				}
			}

	}
}
}

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	// Iterate over all of the Token's attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = string(a.Val)
			ok = true
		}
	}

	// "bare" return will return the variables (ok, href) as defined in
	// the function definition
	return
}
