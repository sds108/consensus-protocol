package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// func Startup() {
// 	reader := bufio.NewReader(os.Stdin)

// 	for serverAddr == nil {
// 		fmt.Print("Please enter the server ip: ")
// 		input, err := reader.ReadString('\n')S

// 		if err != nil {
// 			fmt.Println("Error reading input:", err)
// 			continue
// 		}

// 		input = strings.TrimSpace(input)

// 		// Resolve UDP Server Address to contact
// 		serverAddr, err = net.ResolveUDPAddr("udp", input+":"+SERVER_PORT_CONST)
// 		if err != nil {
// 			log.Fatalf("Failed to resolve server address: %v", err)
// 			fmt.Print(serverAddr)
// 		}
// 	}
// }

func main() {
	Brainloop()
}

func Brainloop() {
	for true {
		//time.Sleep(5 * time.Second)
		ReadInput()
	}
}

// function for reading in the initial strings
func ReadInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n\n--------------------------------------Welcome--------------------------------------\n") //76
	fmt.Print("\nWhat would you like to do?\n\nPlease input the number or the name of the command\nHere is a list of commands: \n\n")
	fmt.Print("0 - 'request vote' \n1 - 'number of clients' \n2 - 'ip of clients' \n3 - 'send with loss' \n4 - 'disconnect' \n")
	fmt.Print("\n-----------------------------------------------------------------------------------\n") //83
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
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("Please input vote request, 'q' to quit writing: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	input = strings.TrimSpace(input)
	if input == "q" {
		ReadInput()
	} else {
		fmt.Print("-----------------------------------------------------------------------------------\n") //83
		fmt.Print("You have chosen to start a vote request.\nProcessing...\n")
		// for _, server := range conversations {
		// 	server.sendVoteRequestToServer(input)
		// 	break
		// }
	}
}

// function entered if a request for client number on network is initiated
func request_client_number() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to request for number of clients on the network.\nProcessing...\n")

	//--------------------------------------------------
	// send to server to request for number of clients on network
	//--------------------------------------------------

}

func request_client_ips() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to request for the ip addresses of the clients on the network.\nProcessing...\n")

	//--------------------------------------------------
	// send to server to request for number of clients on network
	//--------------------------------------------------

}

// function for demonstrating SR maybe the send with loss function?
func request_change_loss() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("Please Choose a decimal value between 0-1\n\n")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	input = strings.TrimSpace(input)
	val, err := strconv.ParseFloat(input, 64)

	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to send packets at a loss of: ", val)

	//--------------------------------------------------
	// change global of send with loss multiplier
	//--------------------------------------------------
}

func request_disconnect() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to disconnect from the server.\n")

	//--------------------------------------------------
	// Send a message to disconnect client from server
	//--------------------------------------------------
	os.Exit(0)
}

// handler of inputs following initial input
func requestHandler(input string) {

	switch input {

	case "0", "request vote":
		request_vote_initiator()

	case "1", "number of clients":
		request_client_number()

	case "2", "ip of clients":
		request_client_ips()

	case "3", "set loss constant":
		request_change_loss()

	case "4", "disconnect":
		request_disconnect()
	}

}
