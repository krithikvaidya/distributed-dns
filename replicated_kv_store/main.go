package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/kv_store"
	"github.com/krithikvaidya/distributed-dns/replicated_kv_store/protos"
	"google.golang.org/grpc"
)

var n_replica int

// Start the local key-value store's HTTP server
func StartKVStore(addr string) {

	kv := kv_store.NewStore() // NewStore() defined in kv_store/restaccess_key_value.go
	r := mux.NewRouter()

	r.HandleFunc("/kvstore", kv.KvstoreHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", kv.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", kv.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", kv.DeleteHandler).Methods("DELETE")

	//Start the server and listen for requests. This is blocking.
	err := http.ListenAndServe(addr, r)

	CheckErrorFatal(err)

}

// Start a server to listen for client requests
func (node *RaftNode) StartRaftServer(addr string) {

	r := mux.NewRouter()

	r.HandleFunc("/test", node.TestHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PostHandler).Methods("POST")
	r.HandleFunc("/{key}", node.GetHandler).Methods("GET")
	r.HandleFunc("/{key}", node.PutHandler).Methods("PUT")
	r.HandleFunc("/{key}", node.DeleteHandler).Methods("DELETE")

	//Start the server and listen for requests. This is blocking.
	err := http.ListenAndServe(addr, r)

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

func main() {

	log.Println("\nRaft-based Replicated Key Value Store\n")

	log.Printf("Enter the replica's id: ")
	var rid int32
	fmt.Scanf("%d", &rid)

	log.Printf("\nEnter the TCP network address that the replica should bind to (eg - :7890): ")
	var address string
	fmt.Scanf("%s", &address)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	CheckErrorFatal(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	CheckErrorFatal(err)

	log.Printf("\nSuccessfully listening at address %v\n", address)

	// Start the local key value store and wait for it to initialize
	var addresskeyvalue string
	log.Printf("\nEnter the port the key-value replica should listen on: ")
	fmt.Scanf("%s", &addresskeyvalue)

	go StartKVStore(addresskeyvalue)

	test_addr := fmt.Sprintf("http://localhost%s/kvstore", addresskeyvalue)

	log.Printf("\nStarting local key-value store...\n")

	// make HTTP request to the test endpoint until a reply is obtained, indicating that
	// the HTTP server is up
	for {

		_, err := http.Get(test_addr)

		if err == nil {
			log.Printf("\nKey-value store up and listening at port %s\n", addresskeyvalue)
			break
		}

	}

	// Get the address to bind the server that listens to client requests.
	log.Printf("\nEnter the port the replica should listen for client requests on: ")
	var server_address string
	fmt.Scanf("%s", &server_address)
	// Starting the server is done after InitializeNode

	log.Printf("\nEnter the addresses of %v other replicas: \n", n_replica-1)

	rep_addrs := make([]string, n_replica)

	for i := int32(0); i < int32(n_replica); i++ {

		if i == rid {
			continue
		}

		fmt.Scanf("%s", &rep_addrs[i])

	}

	grpcServer := grpc.NewServer()

	// InitializeNode() is defined in raft_node.go
	node := InitializeNode(int32(n_replica), rid, addresskeyvalue)

	go node.ApplyToStateMachine() // ApplyToStateMachine defined in raft_node.go

	// Now we can start listening to client requests
	// Start the raft replica server and wait for it to initialize
	go node.StartRaftServer(server_address)

	test_addr = fmt.Sprintf("http://localhost%s/test", server_address)

	log.Printf("\nStarting raft replica server...\n")

	for {

		_, err := http.Get(test_addr)

		if err == nil {
			log.Printf("\nRaft replica server up and listening at port %s\n", server_address)
			break
		}

	}

	// ConsensusService is defined in protos/replica.proto./
	// RegisterConsensusServiceServer is present in the generated .pb.go file
	protos.RegisterConsensusServiceServer(grpcServer, node)

	// gRPC Serve is blocking, so we do it on a separate goroutine
	go func() {

		err := grpcServer.Serve(listener)
		CheckErrorFatal(err)

	}()

	log.Printf("\ngRPC server listening...\n")

	log.Printf("\nPress enter when all other nodes are online.\n")
	var input rune
	fmt.Scanf("%c", &input)

	// Attempt to gRPC dial to other replicas. ConnectToPeerReplicas is defined in raft_node.go
	log.Printf("\nAttempting to connect to peer replicas...\n")
	node.ConnectToPeerReplicas(rep_addrs)
	log.Printf("\nSuccessfully connected to peer replicas.\n") // established connection to all other nodes

	<-node.ready_chan // wait until all connections to our node have been established.

	log.Printf("\nAll peer replicas have successfully connected.\n")

	// dummy channel to ensure program doesn't exit. Remove it later
	all_connected := make(chan bool)
	<-all_connected

}
