package raft

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// Reference for colourization: https://twinnation.org/articles/35/how-to-add-colors-to-your-console-terminal-output-in-go

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func init() {

	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
	}
}

func CheckErrorFatal(err error) {

	if err != nil {
		log.Fatalf(Red + "[Fatal Error]" + Reset + ": " + err.Error()) // The Fatalf functions call os.Exit(1) after writing the log message.
	}

}

func (node *RaftNode) ListenForShutdown(master_cancel context.CancelFunc) {

	// We capture termination signals and ensure that the program shuts down properly.
	os_sigs := make(chan os.Signal, 1)                      // Listen for OS signals, with buffer size 1
	signal.Notify(os_sigs, syscall.SIGTERM, syscall.SIGINT) // SIGKILL and SIGSTOP cannot be caught by a program

	rcvd_sig := <-os_sigs

	log.Printf("\n\nTermination signal received: %v\n", rcvd_sig)

	signal.Stop(os_sigs) // Stop listening for signals
	close(os_sigs)

	master_cancel()

	select {
	case str := <-node.Meta.shutdown_chan:
		log.Printf("\n[1/4] %v", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

	select {
	case str := <-node.Meta.shutdown_chan:
		log.Printf("[2/4] %v", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

	select {
	case str := <-node.Meta.shutdown_chan:
		log.Printf("[3/4] %v", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

	select {
	case str := <-node.Meta.shutdown_chan:
		log.Printf("[4/4] %v\n\n", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

}
