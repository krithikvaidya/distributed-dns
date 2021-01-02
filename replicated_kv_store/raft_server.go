package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		node.raft_node_mutex.RUnlock()
		return
	}

	node.raft_node_mutex.RUnlock()

	value := r.FormValue("value")
	params := mux.Vars(r)
	key := params["key"]

	operation := make([]string, 3)
	operation[0] = "POST"
	operation[1] = key
	operation[2] = value

	if node.WriteCommand(operation) {
		fmt.Fprintf(w, "\nSuccessful POST\n")
	} else {
		fmt.Fprintf(w, "\nPOST request failed to be completed.\n")
	}
}

//handles get requests
func (node *RaftNode) GetHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nGET request received\n")

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		node.raft_node_mutex.RUnlock()
		return
	}

	node.raft_node_mutex.RUnlock()

	params := mux.Vars(r)
	key := params["key"]

	k, _ := strconv.Atoi(key)

	if node.ReadCommand(k) {
		fmt.Fprintf(w, "\nRead success.\n")
	} else {
		fmt.Fprintf(w, "\nRead failed.\n")
	}
}

//handles all put requests
func (node *RaftNode) PutHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nPUT request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		node.raft_node_mutex.RUnlock()
		return
	}

	node.raft_node_mutex.RUnlock()

	value := r.FormValue("value")
	params := mux.Vars(r)
	key := params["key"]

	operation := make([]string, 3)
	operation[0] = "PUT"
	operation[1] = key
	operation[2] = value

	if node.WriteCommand(operation) {
		fmt.Fprintf(w, "\nPUT success.\n")
	} else {
		fmt.Fprintf(w, "\nPUT failed.\n")
	}

}

//handles all delete requests
func (node *RaftNode) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nDELETE request received\n")

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		node.raft_node_mutex.RUnlock()
		return
	}

	node.raft_node_mutex.RUnlock()

	params := mux.Vars(r)
	key := params["key"]

	operation := make([]string, 2)
	operation[0] = "DELETE"
	operation[1] = key

	if node.WriteCommand(operation) {
		fmt.Fprintf(w, "\nDELETE success.\n")
	} else {
		fmt.Fprintf(w, "\nDELETE failed.\n")
	}
}
