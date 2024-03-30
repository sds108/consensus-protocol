package main

import (
	// "fmt"
	// "log"
	// "math/rand"
	"net"
	"os"
	// "sync"
	// "time"
	"bufio"
)


func NewPcktHello(version uint32, features []uint16) *PcktHello{
	return &PcktHello{
		Header:      DataHeader // 8 bytes
		Version:     uint32     // 4 bytes
		NumFeatures: uint16     // 2 bytes
		Feature:     []uint16   // num_features*2 bytes
		//list of connected nodes: []uint16
	}
}

func NewPcktVoteRequest(QuestionLength uint32, Question string) *PcktVoteRequest{
	return & PcktVoteRequest{
		Header:         DataHeader // 8 bytes
		VoteID:         uuid.UUID  // 16 bytes // Golang: https://pkg.go.dev/github.com/beevik/guid Python: https://stackoverflow.com/questions/534839/how-to-create-a-guid-uuid-in-python,
		QuestionLength: uint32     // 4 bytes
		Question:       string     // QuestionLength bytes long, for Z3
	}

}
func Brainloop(){
	Read_Input()
}

func Read_Input(){
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Welcome\nWhat would you like to do?\n please type 'request vote' to start a vote request ")
	input, err := reader.ReadString('\n')
	if err != nil {
		// Handle error, for example, print it and exit
		fmt.Println("Error reading input:", err)
		return
	else 
		request_handler(input)
	}
}

func request_handler( input){
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if input == "request vote" || input == "Request Vote" || input == "Request vote"{
		fmt.Print("Please input vote request or 'escape' to exit")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if input == "escape" {
			return
		}
		else 
		
			//--------------------------------------------------
			//make new vote request with input and send \/\/\/\/\/\/\/\/\/\
			//--------------------------------------------------
	}


}


