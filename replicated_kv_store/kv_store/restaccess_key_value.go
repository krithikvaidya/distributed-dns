package kv_store

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

const (
	length = 101
)

type store struct {
	db       [length]*linkedlist
	mu       sync.RWMutex
	filename string
}

//creates a new instance of key value store
func NewStore(text string) *store {
	return &store{
		filename: text,
	}
}

//test handler
func (kv *store) KvstoreHandler(w http.ResponseWriter, r *http.Request) {

	// if r.URL.Path != "/kvstore" {
	// 	http.Error(w, "404 not found.", http.StatusNotFound)
	// 	return
	// }

	// if r.Method != "GET" {
	// 	http.Error(w, "Method is not supported.", http.StatusNotFound)
	// 	return
	// }

	fmt.Fprintf(w, "KEY VALUE STORE !!!")
}

//handles all post requests
func (kv *store) PostHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nPOST request received\n")

	// if r.Method != "POST" {
	// http.Error(w, "Method is not supported.", http.StatusNotFound)
	// return
	// }

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	kv.mu.Lock()

	value := r.FormValue("value")
	params := mux.Vars(r)
	key := params["key"]

	duplicate := kv.Get(key)
	if duplicate == "Invalid" {
		fmt.Fprintf(w, "Key = %s\n", key)
		fmt.Fprintf(w, "Value = %s\n", value)
		kv.Push(key, value)
	} else {
		fmt.Fprintf(w, "This key already exists")
	}

	kv.mu.Unlock()
	kv.Persist()
}

//handles all get requests
func (kv *store) GetHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nGET request received\n")

	// if r.Method != "GET" {
	// http.Error(w, "Method is not supported.", http.StatusNotFound)
	// return
	// }

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	kv.mu.RLock()

	params := mux.Vars(r)
	key := params["key"]
	value := kv.Get(key)

	if value == "Invalid" {
		fmt.Fprintf(w, "Invalid key value pair\n")
	} else {
		fmt.Fprintf(w, "Value = %s\n", value)
	}

	kv.mu.RUnlock()
}

//handles all put requests
func (kv *store) PutHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nPUT request received\n")

	// if r.Method != "PUT" {
	// http.Error(w, "Method is not supported.", http.StatusNotFound)
	// return
	// }

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	kv.mu.Lock()

	value := r.FormValue("value")
	params := mux.Vars(r)
	key := params["key"]
	ok := kv.Put(key, value)

	if ok == true {
		fmt.Fprintf(w, "Key = %s\n", key)
		fmt.Fprintf(w, "Value = %s\n", value)
	} else {
		fmt.Fprintf(w, "Invalid key value pair\n")
	}

	kv.mu.Unlock()
	kv.Persist()
}

//handles all delete requests
func (kv *store) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nDELETE request received\n")

	// if r.Method != "DELETE" {
	// http.Error(w, "Method is not supported.", http.StatusNotFound)
	// return
	// }

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	kv.mu.Lock()

	params := mux.Vars(r)
	key := params["key"]
	ok := kv.Delete(key)

	if ok == true {
		fmt.Fprintf(w, "Removed Key = %s\n", key)
	} else {
		fmt.Fprintf(w, "Invalid key value pair\n")
	}

	kv.mu.Unlock()
	kv.Persist()
}
