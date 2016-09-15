package main

import (
	"net/http"
	"text/template"
)

type Context struct {
	URL string
}

func launchServer() {
	info("launching server at :8000")
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8000", nil)
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	debug("serving request", req.URL.String())
	tmpl, err := template.New("name").Parse(rootTemplate)
	if err == nil {
		context := Context{"http://localhost:8000/"}
		tmpl.Execute(w, context)

	}
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
