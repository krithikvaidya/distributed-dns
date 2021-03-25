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

func (node *RaftNode) GetLock(where string) {

	// log.Printf("\nReplica %v trying to get lock in %v\n", node.Meta.replica_id, where)
	node.raft_node_mutex.Lock()
	// log.Printf("Replica %v got lock in %v\n", node.Meta.replica_id, where)

}

func (node *RaftNode) ReleaseLock(where string) {

	node.raft_node_mutex.Unlock()
	// log.Printf("\nReplica %v released lock in %v\n", node.Meta.replica_id, where)

}

func (node *RaftNode) GetRLock(where string) {

	// log.Printf("\nReplica %v trying to get RLock in %v\n", node.Meta.replica_id, where)
	node.raft_node_mutex.RLock()
	// log.Printf("\nReplica %v got RLock in %v\n", node.Meta.replica_id, where)

}

func (node *RaftNode) ReleaseRLock(where string) {

	node.raft_node_mutex.RUnlock()
	// log.Printf("\nReplica %v released RLock in %v\n", node.Meta.replica_id, where)

}

// Listen for termination signal and call master cancel. Wait for spawned goroutines to exit.
func (node *RaftNode) ListenForShutdown(master_cancel context.CancelFunc) {

	// We capture termination signals and ensure that the program shuts down properly.
	os_sigs := make(chan os.Signal, 1)                      // Listen for OS signals, with buffer size 1
	signal.Notify(os_sigs, syscall.SIGTERM, syscall.SIGINT) // SIGKILL and SIGSTOP cannot be caught by a program

	rcvd_sig := <-os_sigs

	log.Printf("\n\nTermination signal received: %v\n", rcvd_sig)

	signal.Stop(os_sigs) // Stop listening for signals
	close(os_sigs)

	master_cancel()

	log.Println()
	for i := 1; i <= 4; i++ {

		select {
		case str := <-node.Meta.shutdown_chan:
			log.Printf("[%v/4] %v", i, str)
		case <-time.After(5 * time.Second):
			log.Printf("\nTimeout expired, force shutdown invoked.\n")
			return
		}

	}

}
