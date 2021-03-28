package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
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

type GetEnvResponse struct {
	Message      string   `json:message`
	N_replicas   int      `json:n_replicas`
	Replica_id   int      `json:replica_id`
	Internal_ips []string `json:internal_ips`
	Tg_arn       string   `json:tg_arn`
	Instance_ids []string `json:instance_ids`
}

type EC2InstanceMetadata struct {
	inst_id string
	region  string
}

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

var Inst_meta EC2InstanceMetadata

// Register replica with oracle responsible for assigning it to a nameserver.
func RegisterWithOracle() {

	cmd := exec.Command("wget", "-q", "-O", "-", "http://169.254.169.254/latest/meta-data/instance-id")
	inst_id, err := cmd.Output()

	cmd = exec.Command("ec2metadata", "--availability-zone", "|", "sed 's/.$//'")
	op, _ := cmd.Output()
	region := strings.Trim(string(op), " ")
	region = strings.Trim(string(op), "\n")
	region2 := region[:len(region)-1]

	Inst_meta = EC2InstanceMetadata{

		inst_id: string(inst_id),
		region:  region2,
	}

	req_data := map[string]string{
		"region":  region2,
		"inst_id": string(inst_id),
	}

	jsonValue, _ := json.Marshal(req_data)

	resp, err := http.Post("http://ec2-54-236-50-24.compute-1.amazonaws.com:5000/register_replica", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("%v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v", err)
	}
	bodyString := string(bodyBytes)
	log.Printf(bodyString)
}

func GetEnvFromOracle() {

	req_data := map[string]string{
		"region":  Inst_meta.region,
		"inst_id": Inst_meta.inst_id,
	}

	jsonValue, _ := json.Marshal(req_data)

	for {

		resp, err := http.Post("http://ec2-54-236-50-24.compute-1.amazonaws.com:5000/get_env", "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			fmt.Printf("\nError in GetEnvFromOracle: %v\n", err)
		}

		response_map := &GetEnvResponse{}
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			fmt.Printf("\nError in GetEnvFromOracle: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err = json.Unmarshal(bodyBytes, response_map); err != nil {
			fmt.Printf("\nError in GetEnvFromOracle: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if response_map.Message == "Waiting for more instances to register" {
			fmt.Printf("\nWaiting for more instances to register\n")
			time.Sleep(2 * time.Second)
			continue
		}

		os.Setenv("N_REPLICAS", strconv.Itoa(response_map.N_replicas))
		os.Setenv("REPLICA_ID", strconv.Itoa(response_map.Replica_id))
		os.Setenv("TG_ARN", strings.Trim(strings.Trim(response_map.Tg_arn, " "), "\n"))

		for i := 0; i < response_map.N_replicas; i++ {

			os.Setenv("INST_ID_"+strconv.Itoa(i), strings.Trim(strings.Trim(response_map.Instance_ids[i], " "), "\n"))
			os.Setenv("REP_"+strconv.Itoa(i)+"_INTERNAL_IP", strings.Trim(strings.Trim(response_map.Internal_ips[i], " "), "\n"))

		}

		fmt.Println("Successfully obtained and set environment variables. Env variables are: ")
		fmt.Println(os.Getenv("N_REPLICAS"))
		fmt.Println(os.Getenv("REPLICA_ID"))
		fmt.Println(os.Getenv("TG_ARN"))
		fmt.Println(os.Getenv("INST_0_ID"))
		fmt.Println(os.Getenv("INST_1_ID"))
		fmt.Println(os.Getenv("INST_2_ID"))
		fmt.Println(os.Getenv("REP_0_INTERNAL_IP"))
		fmt.Println(os.Getenv("REP_1_INTERNAL_IP"))
		fmt.Println(os.Getenv("REP_2_INTERNAL_IP"))
		break

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
	iid := "INST_ID_" + strconv.Itoa(int(node.Meta.replica_id))
	target := "Id=" + os.Getenv(iid)

	cmd = exec.Command("aws", "elbv2", "register-targets", "--target-group-arn", tg_arn, "--targets", target)
	out, err = cmd.Output()

	if err != nil {
		log.Printf("Error occured in command2: %v\n", err.Error())
		return
	}

	log.Printf("Output2: %v", out)

}
