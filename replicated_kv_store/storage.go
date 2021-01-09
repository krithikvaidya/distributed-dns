package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"sync"
)

// Storage is a simple in-memory implementation of Storage for testing.
type Storage struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewStorage() *Storage {
	m := make(map[string][]byte)
	return &Storage{
		m: m,
	}
}

func (stored *Storage) writeFile(filename string) {
	dataFile, err := os.Create(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// serialize the data
	dataEncoder := gob.NewEncoder(dataFile)

	stored.mu.Lock()
	defer stored.mu.Unlock()
	dataEncoder.Encode(stored.m)

	dataFile.Close()
}

func (stored *Storage) readFile(filename string) {
	dataFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataDecoder := gob.NewDecoder(dataFile)

	stored.mu.Lock()
	defer stored.mu.Unlock()
	err = dataDecoder.Decode(&stored.m)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataFile.Close()
}

func (stored *Storage) Get(key string, filename string) ([]byte, bool) {
	stored.readFile(filename)
	stored.mu.Lock()
	defer stored.mu.Unlock()
	value, check := stored.m[key]
	return value, check
}

func (stored *Storage) Set(key string, value []byte, filename string) {
	stored.mu.Lock()
	defer stored.mu.Unlock()
	stored.m[key] = value
	stored.writeFile(filename)
}

func (stored *Storage) HasData(filename string) bool {
	stored.readFile(filename)
	stored.mu.Lock()
	defer stored.mu.Unlock()
	return len(stored.m) > 0
}
