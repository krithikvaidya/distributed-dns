package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Test handler
func (node *RaftNode) TestHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "\nServer is up\n\n")

}

//handles all post requests
func (node *RaftNode) PostHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nPOST request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.Lock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nserverAddress: "+node.leaderAddress+"\n") //sends leader address if its not the leader
		node.raft_node_mutex.Unlock()
		return
	}

	value := r.FormValue("value")
	client := r.FormValue("client")
	params := mux.Vars(r)
	key := params["key"]
	fmt.Println(client + "\n")

	operation := make([]string, 3)
	operation[0] = "POST"
	operation[1] = key
	operation[2] = value

	if node.WriteCommand(operation, client) { // Mutex will be unlocked in WriteCommand
		fmt.Fprintf(w, "\nSuccessful POST\n")
	} else {
		fmt.Fprintf(w, "\nPOST request failed to be completed.\n")
	}
}

//handles get requests
func (node *RaftNode) GetHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nGET request received\n")

	// TODO: Make sure all committed entries are applied before responding to it.

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nserverAddress: "+node.leaderAddress+"\n") //sends leader address if its not the leader
		node.raft_node_mutex.RUnlock()
		return
	}

	params := mux.Vars(r)
	key := params["key"]

	if response, err := node.ReadCommand(key); err == nil {

		prnt_str := "\nRead operation completed. Result: " + response + "\n"
		fmt.Fprintf(w, prnt_str)

	} else {

		fmt.Fprintf(w, "\nRead failed with error %v\n", err)

	}
}

//handles all put requests
func (node *RaftNode) PutHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nPUT request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.Lock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nserverAddress: "+node.leaderAddress+"\n")
		node.raft_node_mutex.RUnlock()
		return
	}

	value := r.FormValue("value")
	client := r.FormValue("client")
	params := mux.Vars(r)
	key := params["key"]

	operation := make([]string, 3)
	operation[0] = "PUT"
	operation[1] = key
	operation[2] = value

	if node.WriteCommand(operation, client) {
		fmt.Fprintf(w, "\nPUT success.\n")
	} else {
		fmt.Fprintf(w, "\nPUT failed.\n")
	}

}

//handles all delete requests
func (node *RaftNode) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nDELETE request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.Lock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nserverAddress: "+node.leaderAddress+"\n") //sends leader address if its not the leader
		node.raft_node_mutex.RUnlock()
		return
	}

	params := mux.Vars(r)
	key := params["key"]
	client := r.FormValue("client")

	operation := make([]string, 2)
	operation[0] = "DELETE"
	operation[1] = key

	if node.WriteCommand(operation, client) {
		fmt.Fprintf(w, "\nDELETE success.\n")
	} else {
		fmt.Fprintf(w, "\nDELETE failed.\n")
	}
}
