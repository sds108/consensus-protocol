package main // Declares that this file is part of the main package.

import (
	"fmt" // Import the fmt package for printing.
	"math/rand"
	"net" // Import the net package for networking functionality.
	"strconv"
	"time"
)

func main() { // Defines the main functtion, which is the entry point of the program.
	conn, err := net.Dial("tcp", "172.22.0.2:8080") // Establishes a TCP connection to localhost on port 8080 and returns a connection object and any error encountered
	if err != nil {                                 // Checks if there was an error while dialing
		fmt.Println("Error connecting:", err) // Prints an error message if an error occurred
		return                                // Exits the function
	}

	defer conn.Close() // Defers the closing of the connection until the function returns

	// Send 15 messages then quit
	for i := 0; i < 15; i++ {
		// Send a message to the server
		message := conn.LocalAddr().String() + " " + strconv.Itoa(rand.Intn(1000)) // Defines the message to send to the server.
		_, err = conn.Write([]byte(message))                                       // Writes the mesage to the connection and returns the number of bytes written and any error encountered
		if err != nil {                                                            // Checks if there was an error while writing.
			fmt.Println("Error sending message:", err) // Prints an error message if an error occurred
			return                                     // Exit the function.
		}
		fmt.Println("Sent message to server:", message) // Print a message indicating that the message was sent to the server

		// Receive response from the server
		response := make([]byte, 1024) // Declares buffer to store the response from the server
		n, err := conn.Read(response)  // Reads data from the conection into the buffer and returns the number of bytes read and any error encountered
		if err != nil {                // Checks if there was an error while reading.
			fmt.Println("Error receiving response:", err) // Prints an error message if an error occurred
			return                                        // Exitss the function.
		}
		received := response[:n]                                        // Extracts the received data from the buffer.
		fmt.Println("Received response from server:", string(received)) // Prints the receiveed response from the server.

		time.Sleep(2 * time.Second)
	}

}
