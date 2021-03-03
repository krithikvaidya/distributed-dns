package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
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
	case str := <-node.meta.shutdown_chan:
		log.Printf("\n[1/4] %v", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

	select {
	case str := <-node.meta.shutdown_chan:
		log.Printf("[2/4] %v", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

	select {
	case str := <-node.meta.shutdown_chan:
		log.Printf("[3/4] %v", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

	select {
	case str := <-node.meta.shutdown_chan:
		log.Printf("[4/4] %v\n\n", str)
	case <-time.After(5 * time.Second):
		log.Printf("\nTimeout expired, force shutdown invoked.\n")
		return
	}

}

func (node *RaftNode) LBRegister() {

	log.Println("IN LBRegister")
	// First, deregister all instances from LB (if we try to deregister an unregistered instance, AWS does nothing)

	// Get load balancer ARN from env variable
	lb_arn := os.Getenv("LB_ARN")
	log.Printf("LB_ARN: %v.\n", lb_arn)

	// Get aws instance IDs of all replicas in the cluster
	id0 := os.Getenv("INST_ID_0")
	log.Printf("INST_ID_0: %v.\n", id0)
	id1 := os.Getenv("INST_ID_1")
	log.Printf("INST_ID_1: %v.\n", id1)
	id2 := os.Getenv("INST_ID_2")
	log.Printf("INST_ID_1: %v.\n", id2)

	target0 := "Id=" + id0
	target1 := "Id=" + id1
	target2 := "Id=" + id2

	// Perform deregistration. TODO: error checking for commands?
	cmd := exec.Command("aws", "elbv2", "deregister-targets", "--target-group-arn", lb_arn, "--targets", target0, target1, target2)
	out, err := cmd.Output()

	if err != nil {
		log.Printf("Error occured in command1: %v\n", err.Error())
		return
	}

	log.Printf("Output1: %v", out)

	// Register self
	iid := "INST_ID_" + strconv.Itoa(int(node.meta.replica_id))
	target := "Id=" + os.Getenv(iid)

	cmd = exec.Command("aws", "elbv2", "register-targets", "--target-group-arn", lb_arn, "--targets", target)
	out, err = cmd.Output()

	if err != nil {
		log.Printf("Error occured in command2: %v\n", err.Error())
		return
	}

	log.Printf("Output2: %v", out)

}
