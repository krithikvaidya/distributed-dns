package raft

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/krithikvaidya/distributed-dns/raft/protos"
)

type Storage struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

// Initialise Storage object
func NewStorage() *Storage {

	gob.Register([]protos.LogEntry{})

	m := make(map[string]interface{})
	return &Storage{
		m: m,
	}
}

// Encode as gob and write to file for persistence.
func (stored *Storage) WriteFile(filename string) {
	dataFile, err := os.Create(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// serialize the data
	dataEncoder := gob.NewEncoder(dataFile)

	err = dataEncoder.Encode(stored.m)

	if err != nil {
		log.Printf("Error in WriteFile: %v", err.Error())
	}

	dataFile.Close()
}

// Read the file and decode the gob.
func (stored *Storage) ReadFile(filename string) {
	dataFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataDecoder := gob.NewDecoder(dataFile)

	err = dataDecoder.Decode(&stored.m)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataFile.Close()
}

// Get a particular value from the persisted data.
func (stored *Storage) Get(key string, filename string) (interface{}, bool) {

	stored.mu.RLock()
	defer stored.mu.RUnlock()
	stored.ReadFile(filename)
	value, check := stored.m[key]
	return value, check
}

// Write a particular value to be persisted.
func (stored *Storage) Set(key string, value interface{}) {
	stored.mu.Lock()
	defer stored.mu.Unlock()
	stored.m[key] = value
}

// Check if file where data is persisted has data or not.
func (stored *Storage) HasData(filename string) bool {

	stored.mu.RLock()
	defer stored.mu.RUnlock()

	fi, err := os.Stat(filename)

	if os.IsNotExist(err) {
		return false
	} else if fi.Size() == 0 {
		return false
	}

	return true
}

// Restore persisted Raft state from non volatile memory.
func (node *RaftNode) RestoreFromStorage(storage *Storage) {

	var check bool

	var t1, t2, t3, t4, t5 interface{}

	if t1, check = node.storage.Get("currentTerm", node.Meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but currentTerm not found in storage\n")
	}

	node.currentTerm = t1.(int32)

	if t2, check = node.storage.Get("votedFor", node.Meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but votedFor not found in storage\n")
	}

	node.votedFor = t2.(int32)

	if t3, check = node.storage.Get("log", node.Meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but log not found in storage\n")
	}

	node.log = t3.([]protos.LogEntry)

	if t4, check = node.storage.Get("commitIndex", node.Meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but commitIndex not found in storage\n")
	}

	node.commitIndex = t4.(int32)

	if t5, check = node.storage.Get("lastApplied", node.Meta.raft_persistence_file); !check {
		log.Fatalf("\nFatal: persisted data found, but lastApplied not found in storage\n")
	}

	node.lastApplied = t5.(int32)
}

func (node *RaftNode) PersistToStorage() {

	start := time.Now()

	node.storage.Set("currentTerm", node.currentTerm)
	node.storage.Set("votedFor", node.votedFor)
	node.storage.Set("log", node.log)
	node.storage.Set("commitIndex", node.commitIndex)
	node.storage.Set("lastApplied", node.lastApplied)

	node.storage.WriteFile(node.Meta.raft_persistence_file)

	time_taken := time.Since(start)

	if time_taken > 100*time.Millisecond {
		log.Printf("\nPersisting raft state to storage took time %v", time_taken)
	}

}
