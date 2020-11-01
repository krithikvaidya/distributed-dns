package server

import (

)

type server struct {

}


func Server(pass string, address string) *server {

	// Initialize and return server struct

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

	//  Called from Run()
	// Accept incoming connections from clients and write it to the newConn channel

}


func (ser *server) Run(ctx context.Context) {

	// Bind a socket to a port and start listening on it. Use listenForConnections
	// and listen for new connections written to the channel, and appropriately spawn
	// handleClient. Also handle cancellation of context sent from main.

}
