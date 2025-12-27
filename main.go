package main

import (
	"fmt"
	"net/http"
)

func main() {
	newServerMux := http.NewServeMux()
	newServer := &http.Server{
		Addr:    ":8080",
		Handler: newServerMux,
	}
	newServerMux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	newServerMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	err := newServer.ListenAndServe()
	if err != nil {
		fmt.Printf("server failed to start: %v", err)
	}
}