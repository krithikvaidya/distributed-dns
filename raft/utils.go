package raft

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

func RegisterWithOracle() {

	cmd := exec.Command("wget", "-q", "-O", "-", "http://169.254.169.254/latest/meta-data/instance-id")
	inst_id, err := cmd.Output()

	cmd = exec.Command("ec2metadata", "--availability-zone", "|", "sed 's/.$//'")
	op, _ := cmd.Output()
	region := strings.Trim(string(op), " ")
	region = strings.Trim(string(op), "\n")
	region2 := region[:len(region)-1]

	req_data := map[string]string{
		"region":  region2,
		"inst_id": string(inst_id),
	}

	jsonValue, _ := json.Marshal(req_data)

	resp, err := http.Post("http://ec2-54-236-50-24.compute-1.amazonaws.com:5000/register_replica", "application/json", bytes.NewBuffer(jsonValue))
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v", err)
	}
	bodyString := string(bodyBytes)
	log.Printf(bodyString)
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

func (node *RaftNode) LBRegister() {

	log.Println("IN LBRegister")
	// First, deregister all instances from LB (if we try to deregister an unregistered instance, AWS does nothing)

	// Get target group balancer ARN from env variable
	tg_arn := os.Getenv("TG_ARN")
	log.Printf("TG_ARN: %v.\n", tg_arn)

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
	cmd := exec.Command("aws", "elbv2", "deregister-targets", "--target-group-arn", tg_arn, "--targets", target0, target1, target2)
	out, err := cmd.Output()

	if err != nil {
		log.Printf("Error occured in command1: %v\n", err.Error())
		return
	}

	log.Printf("Output1: %v", out)

	// Register self
	iid := "INST_ID_" + strconv.Itoa(int(node.meta.replica_id))
	target := "Id=" + os.Getenv(iid)

	cmd = exec.Command("aws", "elbv2", "register-targets", "--target-group-arn", tg_arn, "--targets", target)
	out, err = cmd.Output()

	if err != nil {
		log.Printf("Error occured in command2: %v\n", err.Error())
		return
	}

	log.Printf("Output2: %v", out)

}
