package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/kv_store"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

var n_replica int

// Start the local key-value store's HTTP server
func (node *RaftNode) StartKVStore(addr string, num int) {

	filename := "600" + strconv.Itoa(num)

	kv := kv_store.InitializeStore(filename) // NewStore() defined in kv_store/restaccess_key_value.go

	r := mux.NewRouter()

	r.HandleFunc("/kvstore", kv.KvstoreHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", kv.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", kv.DeleteHandler).Methods("DELETE")

	// Create a server struct
	kv_store_server := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	node.node_meta.kv_store_server = kv_store_server

	err := kv_store_server.ListenAndServe()

	CheckErrorFatal(err)

}

// Start a server to listen for client requests
func (node *RaftNode) StartRaftServer(addr string) {

	node.node_meta.nodeAddress = addr //store address of the node

	r := mux.NewRouter()

	r.HandleFunc("/test", node.TestHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", node.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", node.DeleteHandler).Methods("DELETE")

	// Create a server struct
	raft_server := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	node.node_meta.raft_server = raft_server

	//Start the server and listen for requests. This is blocking.
	err := raft_server.ListenAndServe()

	CheckErrorFatal(err)

}

func init() {

	/*
	 * Workaround for a Go bug
	 * The Init() function for the testing package should be called
	 * before our init() function for parsing the command-line arguments
	 * of the `go test` command
	 */
	testing.Init()

	// Command line parameters
	flag.IntVar(&n_replica, "n", 5, "total number of replicas (default=5)")
	flag.Parse()

	log.SetFlags(0) // Turn off timestamps in log output.
	rand.Seed(time.Now().UnixNano())

}

/*
 * This function creates a raft node and imports the persistent
 * state information to the node.
 */
func setup_raft_node(id int, n_replicas int) *RaftNode {

	// Key value store address of the current node
	kv_addr := ":300" + strconv.Itoa(id)

	// Initialize node
	node := InitializeNode(int32(n_replicas), id, kv_addr)

	// Apply the persistent entries to the kv store
	go node.ApplyToStateMachine()

	return node
}

/*
 * This function connects an existing node to a raft system.
 *
 * It connects the current node to the other nodes. This mechanism includes the
 * initiation of their various services, like the
 * KV store server, the gRPC server and the Raft server.
 *
 * Returns -1 in case of an error and 0 in the case of successful execution.
 */
func (node *RaftNode) connect_raft_node(id int, rep_addrs []string, testing bool) int {

	// Starting KV store
	kvstore_addr := ":300" + strconv.Itoa(id)
	log.Println("Starting local key-value store...")
	go node.StartKVStore(kvstore_addr, id)

	/*
	 * Make a HTTP request to the test endpoint until a reply is obtained, indicating that
	 * the HTTP server is up
	 */

	test_addr := fmt.Sprintf("http://localhost%s/kvstore", kvstore_addr)

	if !testing {
		for {

			_, err := http.Get(test_addr)

			if err == nil {
				log.Printf("\nKey-value store up and listening at port %s\n", kvstore_addr)
				break
			}

		}
	}

	/*
	 * Connect the new node to the existing nodes.
	 * Attempt to gRPC dial to other replicas and obtain corresponding client stubs.
	 * ConnectToPeerReplicas is defined in raft_node.go.
	 */
	log.Println("Obtaining client stubs of gRPC servers running at peer replicas...")
	node.ConnectToPeerReplicas(rep_addrs)

	// Setting up and running the gRPC server
	grpc_address := ":500" + strconv.Itoa(id)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", grpc_address)
	CheckErrorFatal(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	CheckErrorFatal(err)

	node.node_meta.grpc_server = grpc.NewServer()

	/*
	 * ConsensusService is defined in protos/replica.proto./
	 * RegisterConsensusServiceServer is present in the generated .pb.go file
	 */
	protos.RegisterConsensusServiceServer(node.node_meta.grpc_server, node)

	// Running the gRPC server
	go func() {
		err := node.node_meta.grpc_server.Serve(listener)
		CheckErrorFatal(err)
		log.Printf("\ngRPC server successfully listening at address %v\n", grpc_address)
	}()

	// TODO: wait till grpc server up

	// Now we can start listening to client requests

	// Set up the server for the Raft functionalities
	server_address := ":400" + strconv.Itoa(id)
	log.Println("Starting raft replica server...")
	go node.StartRaftServer(server_address)

	test_addr = fmt.Sprintf("http://localhost%s/test", server_address)

	// Check whether the server is active
	if !testing {
		for {

			_, err = http.Get(test_addr)

			if err == nil {
				log.Printf("\nRaft replica server up and listening at port %s\n", server_address)
				break
			}

		}
	}

	return 0
}

func main() {

	log.Println("Raft-based Replicated Key Value Store")

	// Get replica id from env variable
	rid, _ := strconv.Atoi(os.Getenv("REPLICA_ID"))
	log.Printf("Replica ID = %v\n", rid)

	node := setup_raft_node(rid, n_replica)

	// Store the gRPC address of other replicas
	rep_addrs := make([]string, n_replica)
	for i := 0; i < n_replica; i++ {
		if i == rid {
			continue
		}

		rep_i_addr := "REP_" + strconv.Itoa(i) + "_GRPC_ADDR"
		rep_addrs[i] = os.Getenv(rep_i_addr)
	}

	if node.connect_raft_node(rid, rep_addrs, false) == 0 {
		log.Println("Node initialization procedure completed")
	} else {
		log.Println("Node initialization procedure failed")
	}

	// dummy channel to ensure program doesn't exit. Remove it later
	all_connected := make(chan bool)
	<-all_connected

}
