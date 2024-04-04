package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func Startup() {
	reader := bufio.NewReader(os.Stdin)

	for true {
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
			fmt.Print(serverAddr)
		} else {
			break
		}
	}
}

func Brainloop() {
	for true {
		ReadInput()
	}
}

func printWelcome() {
	fmt.Print("\n\n--------------------------------------Welcome--------------------------------------\n") //76
	fmt.Print("\nWhat would you like to do?\n\nPlease input the number or the name of the command\nHere is a list of commands: \n")
	//fmt.Print("0 - 'request vote' \n1 - 'number of clients' \n2 - 'ip of clients' \n3 - 'send with loss' \n4 - 'disconnect' \n")
	fmt.Print("\n0 - 'help'\n1 - 'request vote'\n2 - 'send with duplicates'\n3 - 'send with loss'\n4 - 'set chance of defect'\n5 - 'send hello'\n6 - 'disconnect'\n")
	fmt.Print("\n-----------------------------------------------------------------------------------\n") //83
}

// function for reading in the initial strings
func ReadInput() {
	reader := bufio.NewReader(os.Stdin)
	printWelcome()
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
		for _, server := range conversations {
			server.sendVoteRequestToServer(input)
			break
		}
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

// function for demonstrating defection and consensus
func request_defect_rate() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("Please Choose a decimal value between 0-1\n\n")

	var val float64
	for true {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		input = strings.TrimSpace(input)
		val, err = strconv.ParseFloat(input, 64)
		if err != nil || val < 0 || val > 1 {
			fmt.Println("Make sure your value is between 0 and 1")
		} else {
			break
		}
	}

	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to defect to votes at a rate of: ", val)

	//--------------------------------------------------
	// change global of send with loss multiplier
	//--------------------------------------------------

	defect_constant = val
}

// function for demonstrating SR maybe the send with loss function
func request_change_loss() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("Please Choose a decimal value between 0-1\n\n")

	var val float64
	for true {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		input = strings.TrimSpace(input)
		val, err = strconv.ParseFloat(input, 64)
		if err != nil || val < 0 || val > 1 {
			fmt.Println("Make sure your value is between 0 and 1")
		} else {
			break
		}
	}

	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to send packets at a loss rate of: ", val)

	//--------------------------------------------------
	// change global of send with loss multiplier
	//--------------------------------------------------

	loss_constant = val
}

// function for demonstrating defection and consensus
func request_duplicates() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("Please Choose a integer value between 0-255\n\n")

	var val uint64
	for true {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		input = strings.TrimSpace(input)
		val, err = strconv.ParseUint(input, 10, 8)
		if err != nil || val < 0 || val > 255 {
			fmt.Println("Make sure your value is between 0 and 255")
		} else {
			break
		}
	}

	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("You have chosen to send packets with a this many duplicates: ", val)

	//--------------------------------------------------
	// change global of send with loss multiplier
	//--------------------------------------------------

	duplicates_mode = val
}

func request_send_hello() {
	fmt.Print("-----------------------------------------------------------------------------------\n") //83
	fmt.Print("Sending Hello...\n")

	for _, server := range conversations {
		server.sendHello()
		break
	}
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

	case "0", "help":
		printWelcome()

	case "1", "request vote":
		request_vote_initiator()

	// case "1", "number of clients":
	// 	request_client_number()

	// case "2", "ip of clients":
	// 	request_client_ips()

	case "2", "send with duplicates":
		request_duplicates()

	case "3", "set loss constant":
		request_change_loss()

	case "4", "set chance of defect":
		request_defect_rate()

	case "5", "send hello":
		request_send_hello()

	case "6", "disconnect":
		request_disconnect()
	}

}
