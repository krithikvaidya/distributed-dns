/**
This file contains the definition for the server struct and the functions it implements.

There are two main classes of goroutines that exist in the server process:
- The first class is a single goroutine that listens for connection requests from clients.
  Whenever it receives a connection request, it creates a new goroutine that would come under
  the second class.
- The second class contains goroutines that handles the already established connections between
  the clients. These goroutines allow the server and the client to send messages to each other.
  Whenever a message is received from a particular client, the recipient's address is determined
  using the map containing the current connections. The message is then sent to the relevant client
  through the goroutine corresponding to that client.
*/

package server

import (
	"context"
	"net"
)

// This struct characterizes the server process
type server struct {
}

func Server(pass string, address string) *server {

	/*
		An instance of the 'server' struct is created, initialized with given
		or the default data(if the user hasn't specified the data).
	*/

}

func (ser *server) listenForMessages(ctx context.Context, conn net.Conn, username string, term chan bool) {

	/**
	Method parameter description:
	1. ctx - cancellable context
	2. conn - represents the socket connection to the client
	2. username - client username
	3. term - write to this channel on receiving termination request from client
	*/

	/**
	Spawned by handleClient(). Listens for messages sent by the given client, and appropriately unicasts/broadcast/
	prints error message, etc.
	*/

}

func (ser *server) handleClient(ctx context.Context, conn net.Conn) {

	/**
	Spawned by Run() when a client connection is received. Performs authentication and username checking, responds
	appropriately and after that, uses listenForMessages to handle incoming messages. Should handle cancellation of context
	and messages written to the term channel in the above function
	*/

}

func (ser *server) listenForConnections(ctx context.Context, newConn chan net.Conn, listener *net.TCPListener) {

	// Called from Run()
	// Accept incoming connections from clients and write it to the newConn channel

}

func (ser *server) Run(ctx context.Context) {

	// Bind a socket to a port and start listening on it. Use listenForConnections
	// and listen for new connections written to the channel, and appropriately spawn
	// handleClient. Also handle cancellation of context sent from main.

}
