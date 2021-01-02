package raft_server

import (
	"fmt"
	"net/http"
)

// Test handler
func TestHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "\nServer is up\n\n")
}
