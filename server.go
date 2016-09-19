package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"
)

type Context struct {
	URL string
}

type Test struct {
	URL     string
	start   int64
	end     int64
	success bool
	name    string
}

func launchServer() {
	info("launching server at :8000")
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8000", nil)
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	result, diff := politenessTest(req)
	if !result {
		fmt.Println("politeness failed by", strconv.Itoa(int(diff)))
	}
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
	result := true
	niltime := time.Time{}
	var diff int64 = 5
	if timestamp == niltime {
		timestamp = time.Now()
	} else {
		diff = time.Now().Unix() - timestamp.Unix()
		if diff < politeness {
			result = false
			timestamp = time.Now()
		} else {
			result = true
		}
	}

	return result, diff
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
