package main

import (

	"flag"
	...

)

// Global variable declarations
var (
	serverMode bool
	debugMode  bool
	host       string
	password   string
	username   string
)

func init() {

	// Initialize global variables using command-line arguments using the flag package.

}

func init() {

	// Initialize random seed 

}

func init() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(0)
}
