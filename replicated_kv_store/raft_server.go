package main

import (
	"fmt"
	"net/http"
)

// Test handler
func (node *RaftNode) TestHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "\nServer is up\n\n")
}

//handles all post requests
func (node *RaftNode) PostHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nPOST request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.RLock()
	defer node.raft_node_mutex.RUnlock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		return
	}

	// value := r.FormValue("value")
	// params := mux.Vars(r)
	// key := params["key"]

	// ...
}

//handles get requests
func (node *RaftNode) GetHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nGET request received\n")

	node.raft_node_mutex.RLock()
	defer node.raft_node_mutex.RUnlock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		return
	}

	// params := mux.Vars(r)
	// key := params["key"]

	// ...
}

//handles all put requests
func (node *RaftNode) PutHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nPUT request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.RLock()
	defer node.raft_node_mutex.RUnlock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		return
	}

	// value := r.FormValue("value")
	// params := mux.Vars(r)
	// key := params["key"]

	// ...
}

//handles all delete requests
func (node *RaftNode) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nDELETE request received\n")

	node.raft_node_mutex.RLock()
	defer node.raft_node_mutex.RUnlock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		return
	}

	// params := mux.Vars(r)
	// key := params["key"]

	// ...
}
