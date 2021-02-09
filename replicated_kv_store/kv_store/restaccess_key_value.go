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
	db       [length]*Linkedlist
	mu       sync.RWMutex
	filename string
	db_temp  map[string]string
}

//creates a new instance of key value store
func InitializeStore(text string) *store {

	kv := &store{
		filename: text,
		db_temp:  make(map[string]string),
	}

	if kv.HasData() {
		kv.Recover()
	}

	return kv
}

//test handler
func (kv *store) KvstoreHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "KEY VALUE STORE !!!")
}

//handles all post requests
func (kv *store) PostHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nPOST request received\n")

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
		kv.db_temp[key] = value
	} else {
		fmt.Fprintf(w, "This key already exists")
	}

	kv.Persist()
	kv.mu.Unlock()
}

//handles all get requests
func (kv *store) GetHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nGET request received\n")

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
		kv.db_temp[key] = value
	} else {
		fmt.Fprintf(w, "Invalid key value pair\n")
	}

	kv.Persist()
	kv.mu.Unlock()
}

//handles all delete requests
func (kv *store) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("\nDELETE request received\n")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	kv.mu.Lock()

	params := mux.Vars(r)
	key := params["key"]
	ok := kv.Delete(key)

	if ok == true {
		fmt.Fprintf(w, "Removed Key = %s\n", key)
		kv.db_temp[key] = ""
	} else {
		fmt.Fprintf(w, "Invalid key value pair\n")
	}

	kv.Persist()
	kv.mu.Unlock()
}
