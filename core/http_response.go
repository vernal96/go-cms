package core

import "net/http"

type HTTPResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func HTMLResponse(html string) HTTPResponse {
	return HTTPResponse{
		StatusCode:  http.StatusOK,
		ContentType: "text/html; charset=utf-8",
		Body:        []byte(html),
	}
}
