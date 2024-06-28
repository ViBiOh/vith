package main

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httputils"
)

func newPort(clients clients, services services) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("HEAD /", services.vith.HandleHead)
	mux.HandleFunc("GET /", services.vith.HandleGet)
	mux.HandleFunc("POST /", services.vith.HandlePost)
	mux.HandleFunc("PUT /", services.vith.HandlePut)
	mux.HandleFunc("PATCH /", services.vith.HandlePatch)
	mux.HandleFunc("DELETE /", services.vith.HandleDelete)

	return httputils.Handler(mux, clients.health,
		clients.telemetry.Middleware("http"),
	)
}
