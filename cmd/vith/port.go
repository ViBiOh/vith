package main

import "net/http"

func newPort(service *service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("HEAD /", service.vith.HandleHead)
	mux.HandleFunc("GET /", service.vith.HandleGet)
	mux.HandleFunc("POST /", service.vith.HandlePost)
	mux.HandleFunc("PUT /", service.vith.HandlePut)
	mux.HandleFunc("PATCH /", service.vith.HandlePatch)
	mux.HandleFunc("DELETE /", service.vith.HandleDelete)

	return mux
}
