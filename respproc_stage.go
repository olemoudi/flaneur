package main

import (
	"net/http"
	"path"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const extractorCount = 10

func responseProcessor(id int) {
	defer wg.Done()
	for i := 1; i <= extractorCount; i++ {
		wg.Add(1)
		go linkExtractor()
	}

	// TODO: can we remove all this?
loop:
	for {
		select {
		case <-exiting:
			debug("processor exiting")
			break loop
		case <-time.After(time.Second * 500):
			//ping()
			continue loop // TODO: here goes the fan out

		}
	}
	return
}

func linkExtractor() {
	defer wg.Done()
loop:
	for {
		select {
		case <-exiting:
			debug("link extractor exiting")
			break loop
		case resp := <-downloadOutputQ:
			ping()
			parseHTML(resp)
			//regexResponse(resp)

		}
	}
	return

}

func parseHTML(resp *http.Response) {
	debug("extracting links")

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		debug("error creating goquery document")
		return
	}
	defaultScheme := resp.Request.URL.Scheme
	defaultAuthority := resp.Request.URL.Host
	defaultPath := path.Dir(resp.Request.URL.EscapedPath()) + "/"

	// use CSS selector found with the browser inspector
	// for each, use index and item
	doc.Find("body a").Each(func(index int, item *goquery.Selection) {
		linkTag := item
		link, _ := linkTag.Attr("href")

		link = getFullLink(link, defaultScheme, defaultAuthority, defaultPath)
		//debug(link)
		req, err := http.NewRequest("GET", link, nil)
		if err == nil {
			select {
			case reqFilterInputQ <- req:
			default:
			}
		}
	})
}

var fullLink = regexp.MustCompile("(?i)^https?://.*$")
var inheritScheme = regexp.MustCompile("^//.*$")
var absoluteLink = regexp.MustCompile("^/[^/].*$")
var relativeLink = regexp.MustCompile("^[^/].*$")

func getFullLink(link string, defaultScheme string, defaultAuthority string, defaultPath string) string {
	switch {
	case fullLink.MatchString(link):
		return link
	case inheritScheme.MatchString(link):
		return defaultScheme + "://" + link + url
	case absoluteLink.MatchString(link):
		return defaultScheme + "://" + defaultAuthority + link
	case relativeLink.MatchString(link):
		return defaultScheme + "://" + defaultAuthority + defaultPath + link
	}
	return link
}

/*

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
*/
