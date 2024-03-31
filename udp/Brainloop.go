package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	//"github.com/google/uuid"
)

// Assuming DataHeader is an existing type; using a placeholder struct for demonstration.
// type DataHeader struct{}

// Placeholder struct based on the provided constructor.
// type PcktHello struct {
// 	Header      DataHeader
// 	Version     uint32
// 	NumFeatures uint16
// 	Feature     []uint16
// }

// Assuming a placeholder struct based on the provided constructor.
// type PcktVoteRequest struct {
// 	Header         DataHeader
// 	VoteID         uuid.UUID
// 	QuestionLength uint32
// 	Question       string
// }

// func NewPcktHello(version uint32, features []uint16) *PcktHello {
// 	return &PcktHello{
// 		Header:      DataHeader{},
// 		Version:     version,
// 		NumFeatures: uint16(len(features)),
// 		Feature:     features,
// 	}
// }

// func NewPcktVoteRequest(QuestionLength uint32, Question string) *PcktVoteRequest {
// 	return &PcktVoteRequest{
// 		Header:         DataHeader{},
// 		VoteID:         uuid.New(), // Generate a new UUID
// 		QuestionLength: QuestionLength,
// 		Question:       Question,
// 	}
// }


func main() {
	Brainloop()
}

func Brainloop() {
	ReadInput()
}

func ReadInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Welcome\nWhat would you like to do?\nHere is a list of commands: \n ")
	fmt.Print("'request vote' \n 'number of clients' \n 'ip of clients' \n 'send with loss' \n")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	// The input from ReadString includes a newline character; we need to trim it
	input = strings.TrimSpace(input)
	requestHandler(input)
}

func request_vote_initiator(){
	fmt.Print("Please input vote request, 'return' to go back or 'quit' to exit: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		input = strings.TrimSpace(input)
		if input == "return"{
			ReadInput()
		}

		if input == "quit" {
			return
		} else {
			fmt.Print("You have chosen to start a vote request.\nProcessing...\n")

			//--------------------------------------------------
			// Make new vote request with input and send
			//--------------------------------------------------
		}
}

func request_client_number(){
	fmt.Print("You have chosen to request for number of clients on the network.\nProcessing...\n")

	//--------------------------------------------------
	// send to server to request for number of clients on network
	//--------------------------------------------------

}

func request_client_ips(){
	fmt.Print("You have chosen to request for the ip addresses of the clients on the network.\nProcessing...\n")

	//--------------------------------------------------
	// send to server to request for number of clients on network
	//--------------------------------------------------

}

func request_send_with_loss(){
	fmt.Print("You have chosen to request for sending packets with loss to the server.\nProcessing...\n")

	//--------------------------------------------------
	// Ken's send with loss function?
	//--------------------------------------------------
}

func requestHandler(input string) {


	if input == "request vote" {
		request_vote_initiator()
	}

	if input == "number of clients"{
		request_client_number()
	}
	if input == "ip of clients"{
		request_client_ips()
	}
	
	if input == "send with loss"{
		request_send_with_loss()
	}
}
