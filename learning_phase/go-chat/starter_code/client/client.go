/**
This file contains the definition for the client struct and functions implemented
by the client struct.

Other related functions are also contained in this file

There are two main phases that a client process goes through:
- The first phase is an initialization procedure where the client object is created and
the connection with the target server is established.
- The second phase begins after the connection with the server is established. This phase
is where the main messaging loop is maintained.

Functionally, this messaing loop of the client process is comprised of two goroutines,
one listening for and receiving messages from the server it is connected to and the other,
that takes the message from the user and sends it to the server it is connected to.
*/

package client

import (


)

/*
This is the struct that characterizes each client process
*/
type client struct {

	Username       string  // If not specified, a random string is chosen using RandSeq() in shared.go
	ServerPassword string  // The server password specified by the user
	ServerHost     string  // ip:port or :port, by default, it is "0.0.0.0:4545"

}


type message_struct struct {

	Type    int  // can define 0 = unicast, 1 = broadcast
	Message string  // The string containing the message
	Sender  string  // The username of the sender

}


func Client(password string, host string, username string) *client {

	/*
	An instance of the 'client' struct is created, initialized with given
	or the default data(if the user hasn't specified the data).
	*/

}

func getServerMessage(conn net.Conn, rcvd_msg chan string, exit_chan chan bool) {

	/**
	Function parameter description:
	1. conn - represents the socket connection to the server
	2. rcvd_msg - on receiving a message from the server, write to this channel
	3. exit_chan - if the server sends a termination signal (eg. writes EOF), write to this channel
	*/

	/** 
	Called by listenForServerMessages defined below.
	Runs an infinite for loop to read messages sent by server
	TCP guarantees reliable and ordered delivery but does not guarantee that the entire message (256 bytes in
	our case) will be delivered in the same packet. So, the it needs to wait until the entire 256 bytes is read
	(or an EOF is received, which will cause an error). Then it should check for errors and handle them appropriately.
	Finally it should write the message as a string (instead of a byte array) to the rcvd_msg channel.
	*/

}

func (cli *client) listenForServerMessages(ctx context.Context, conn net.Conn, msg_channel chan message_struct, term_chan chan bool, final_term_chan chan bool) {

	/**
	Method parameter description:
	1. ctx - cancellable context
	2. conn - represents the socket connection to the server
	3. msg_channel - the Run() function will be listening for messages on this channel
	4. term_chan - when the server sends a termination message, write to this channel to inform Run() (which
	   is monitoring messages sent on this channel also) to exit
	5. final_term_chan - write to this channel when the function has completely exited to inform Run()
	*/
	
	/**
	Called by Run(). Uses getServerMessage to listen for messages. When a message has been received, it should be parsed
	so that it can be printed to the user, and finally printed in this function itself or by sending it to Run() using
	msg_channel (in the latter case msg_channel wouldn't be required). Should also be able to receive termination signals
	from getServerMessage and cancellation of context from Run(), and terminate accordingly.
	*/


}


func getClientMessage(sc bufio.Scanner, conn net.Conn, rcvd_msg chan string) {

	/**
	Called by listenForClientMessages defined below. Should listen for messages entered by the user and
	write it to the rcvd_msg channel
	*/

}

func (cli *client) listenForClientMessages(ctx context.Context, sc bufio.Scanner, conn net.Conn, final_term_chan chan bool) {

	/**
	Called by Run(). Uses getClientMessage to read messages entered by the user. When a message has been read, it should
	be parsed and be made to follow the protocol format, and sent to the server. Should be able to handle cancellation of context
	from Run() too.
	*/

}

func (cli *client) Run(ctx context.Context, main_term_chan chan bool) {

	/**
	Function parameter description:
	1. ctx - cancellable context passed from main
	2. main_term_chan - write to this channel when the function has completely exited to inform main
	*/

	/** 
	Called by main.go
	Attempt to create a TCP connection to the server, authenticate client and check for errors returned by server.
	If there are no errors, create a Scanner to read user inputs (you can use NewScanner in bufio package).
	Then you will need to listen for user input as well as messages delivered by the server, using the above defined functions.
	Note: there are two ways through which the program termination will occur - when the user exits the program 
	(the signal will be caught by main() and propogated to this function) as well as if the server tells the client to terminate
	for whatever reason. You should listen for and handle both.
	*/


}