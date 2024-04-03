package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func Startup() {
	reader := bufio.NewReader(os.Stdin)

	for serverAddr == nil {
		fmt.Print("Please enter the server ip: ")
		input, err := reader.ReadString('\n')

		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)

		// Resolve UDP Server Address to contact
		serverAddr, err = net.ResolveUDPAddr("udp", input+":"+SERVER_PORT_CONST)
		if err != nil {
			log.Fatalf("Failed to resolve server address: %v", err)
			continue
		}
	}
}

// func main() {
// 	Brainloop()
// }

func Brainloop() {

	for true {
		time.Sleep(5 * time.Second)
		ReadInput()
	}
}

// function for reading in the initial strings
func ReadInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Welcome\nWhat would you like to do?\nHere is a list of commands: \n ")
	fmt.Print("'request vote' \n 'number of clients' \n 'ip of clients' \n 'send with loss' \n 'disconnect' \n")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	// The input from ReadString includes a newline character; we need to trim it
	input = strings.TrimSpace(input)
	requestHandler(input)
}

// function entered if a vote request is initiated
func request_vote_initiator() {
	fmt.Print("Please input vote request, 'return' to go back or 'disconnect' to exit: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	input = strings.TrimSpace(input)
	if input == "return" {
		ReadInput()
	}

	if input == "disconnect" {
		request_disconnect()

	} else {
		fmt.Print("You have chosen to start a vote request.\nProcessing...\n")
		for _, server := range conversations {
			server.sendVoteRequestToServer(input)
			break
		}
	}
}

// function entered if a request for client number on network is initiated
func request_client_number() {
	fmt.Print("You have chosen to request for number of clients on the network.\nProcessing...\n")

	//--------------------------------------------------
	// send to server to request for number of clients on network
	//--------------------------------------------------

}

func request_client_ips() {
	fmt.Print("You have chosen to request for the ip addresses of the clients on the network.\nProcessing...\n")

	//--------------------------------------------------
	// send to server to request for number of clients on network
	//--------------------------------------------------

}

// function for demonstrating SR maybe the send with loss function?
func request_send_with_loss() {
	fmt.Print("You have chosen to request for sending packets with loss to the server.\nProcessing...\n")

	//--------------------------------------------------
	// change global boolean of send with loss
	//--------------------------------------------------
}

func request_send_without_loss() {
	fmt.Print("You have chosen to request for sending packets without loss to the server.\nProcessing...\n")

	//--------------------------------------------------
	// change global boolean of send without loss
	//--------------------------------------------------

}

func request_disconnect() {
	fmt.Print("You have chosen to disconnect from the server.\n")

	//--------------------------------------------------
	// Send a message to disconnect client from server
	//--------------------------------------------------
	os.Exit(0)
}

// handler of inputs following initial input
func requestHandler(input string) {

	if input == "request vote" {
		request_vote_initiator()
	}

	if input == "number of clients" {
		request_client_number()
	}
	if input == "ip of clients" {
		request_client_ips()
	}

	if input == "send with loss" {
		request_send_with_loss()
	}

	if input == "send without loss" {
		request_send_without_loss()

	}

	if input == "disconnect" {
		request_disconnect()

	}
}
