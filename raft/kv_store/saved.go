package kv_store

import (
	"encoding/gob"
	"fmt"
	"os"
)

func (kv *store) writeFile() {

	dataFile, err := os.Create(kv.filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// serialize the data
	dataEncoder := gob.NewEncoder(dataFile)

	// CHECK: why is "runtime error: invalid memory address or nil pointer dereference"
	// happening here everytime the Encode happens, even though the write actually succeeds
	// err = dataEncoder.Encode(kv.db_temp)
	// log.Printf("Error in encoding data: %v", err.Error())

	dataEncoder.Encode(kv.db_temp)

	dataFile.Close()
}

func (kv *store) readFile() {

	dataFile, err := os.Open(kv.filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataDecoder := gob.NewDecoder(dataFile)

	err = dataDecoder.Decode(&kv.db_temp)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dataFile.Close()

	for key, value := range kv.db_temp {

		if value != "" {
			kv.Push(key, value)
		}

	}
}

func (kv *store) Recover() {

	kv.mu.RLock()
	defer kv.mu.RUnlock()
	kv.readFile()

}

func (kv *store) Persist() {
	kv.writeFile()
}

func (kv *store) HasData() bool {

	kv.mu.RLock()
	defer kv.mu.RUnlock()

	fi, err := os.Stat(kv.filename)

	if os.IsNotExist(err) {
		return false
	} else if fi.Size() == 0 {
		return false
	}

	return true
}
