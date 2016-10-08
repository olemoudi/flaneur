package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

type Context struct {
	Path string
}

type Test struct {
	Path      string
	Name      string
	Handler   func(http.ResponseWriter, *http.Request)
	Validator func(http.ResponseWriter, *http.Request)
}

func launchServer() {
	info("launching server at :8000")
	http.HandleFunc("/", globalHandler(rootHandler))
	//http.HandleFunc("/test/scope", globalHandler(scopeTest))
	http.ListenAndServe(":8000", nil)
}

func globalHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pt, diff := politenessTest(r)
		if pt {
			fmt.Println("Politeness Test failed by", strconv.Itoa(int(diff)))
		}
		if dupeTest(r) {
			fmt.Println("URL visited twice:", r.URL.String())
		}
		fn(w, r)
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	tmpl, err := template.New("name").Parse(rootTemplate)
	if err == nil {
		context := Context{"/"}
		tmpl.Execute(w, context)

	}
}

/*
func makeTestHandler(fn func(req *http.Request) (bool, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Here we will extract the page title from the Request,
		// and call the provided handler 'fn'
	}
}
*/

var timestamp time.Time = time.Time{}
var politeness int64 = 5
var mutex = &sync.Mutex{}

func politenessTest(req *http.Request) (bool, int64) {
	mutex.Lock()
	defer mutex.Unlock()
	result := false
	niltime := time.Time{}
	var diff int64 = 5
	if timestamp == niltime {
		timestamp = time.Now()
	} else {
		diff = time.Now().Unix() - timestamp.Unix()
		if diff < politeness {
			result = true
			timestamp = time.Now()
		} else {
			result = false
		}
	}

	return result, diff
}

var seen map[string]interface{} = make(map[string]interface{})

func dupeTest(r *http.Request) bool {
	u := strings.TrimSpace(r.URL.String())
	_, dup := seen[u]
	if dup {
		return true
	}
	seen[u] = struct{}{}
	return false
}

const rootTemplate = `
<!DOCTYPE html>
<html>
<head lan="en">
    <meta charset="UTF-8">
    <title>Root Template</title>
</head>
<body>
    <a href="{{.URL}}">link</a>
</body>
</html>
`
