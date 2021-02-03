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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.RLock()

	if node.state != Leader {

		fmt.Fprintf(w, "\nError: Not a leader.\n")

		fmt.Fprintf(w, "\nLast known leader's address: "+node_meta.leaderAddress+"\n") //sends leader address if its not the leader
		node.raft_node_mutex.RUnlock()
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

	success, err := node.WriteCommand(operation, client)
	if success { // Mutex will be unlocked in WriteCommand
		log.Printf("\nPOST request completed successfully and committed.\n")
		fmt.Fprintf(w, "\nPOST request completed successfully and committed.\n")
	} else {
		log.Printf("\n%v\n", err.Error())
		fmt.Fprintf(w, "\n%v\n", err.Error())
	}
}

//handles get requests
func (node *RaftNode) GetHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nGET request received\n")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// TODO: Make sure all committed entries are applied before responding to it.

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nLast known leader's address: "+node_meta.leaderAddress+"\n") //sends leader address if its not the leader
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nLast known leader's address: "+node_meta.leaderAddress+"\n")
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

	success, err := node.WriteCommand(operation, client)
	if success { // Mutex will be unlocked in WriteCommand
		log.Printf("\nPUT request completed successfully and committed.\n")
		fmt.Fprintf(w, "\nPUT request completed successfully and committed.\n")
	} else {
		log.Printf("\n%v\n", err.Error())
		fmt.Fprintf(w, "\n%v\n", err.Error())
	}

}

//handles all delete requests
func (node *RaftNode) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nDELETE request received\n")

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	node.raft_node_mutex.RLock()

	if node.state != Leader {
		fmt.Fprintf(w, "\nError: Not a leader.\n")
		fmt.Fprintf(w, "\nLast known leader's address: "+node_meta.leaderAddress+"\n") //sends leader address if its not the leader
		node.raft_node_mutex.RUnlock()
		return
	}

	params := mux.Vars(r)
	key := params["key"]

	operation := make([]string, 2)
	operation[0] = "DELETE"
	operation[1] = key

	success, err := node.WriteCommand(operation, "")
	if success { // Mutex will be unlocked in WriteCommand
		log.Printf("\nDELETE requested completed successfully and committed.\n")
		fmt.Fprintf(w, "\nDELETE requested completed successfully and committed.\n")
	} else {
		log.Printf("\n%v\n", err.Error())
		fmt.Fprintf(w, "\n%v\n", err.Error())
	}
}
