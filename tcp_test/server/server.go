package main // Declares that this file is part of the main package.

import (
	"fmt" // Import the fmt package for printing.
	"net" // Import the net package for networking functionality.
)

// This defines the handle function to handle each connection to the server.
func handleConnection(conn net.Conn) { // Defines a function named handleConnection that takes a net.Conn object as input
	defer conn.Close()                                           // Defers closing of connection until the function returns
	fmt.Println("Accepted connection from: ", conn.RemoteAddr()) // Prints a message indicating a connection was accepted and the remote address

	buffer_full := true
	for buffer_full {

		// Read data from the client
		buffer := make([]byte, 1024) // Declares a buffer in order to store incoming data
		n, err := conn.Read(buffer)  // Reads the data from the connection into the buffer and returns the number of bytes read and any error encountered
		if err != nil {              // Checks if there was an erro r while reading
			fmt.Println("Error reading:", err) // Prints an error message if an error occurred.
			return                             // Exits the function.
		}
		received := buffer[:n]                              // Extracts the received data from the buffer
		fmt.Println("Received message: ", string(received)) // Prints the received message.

		// Echo back to the client
		_, err = conn.Write(received) // Writes the received data back to the client and returns the number of bytes written and any error encountered
		if err != nil {               // Checks if there was an error while writing
			fmt.Println("Error writing: ", err) // Prints error message if an error occurred.
			return                              // Exits the function
		}
		//fmt.Println("Sent message back to client") // Prints a message indicating that the message was sent back to the client

		if n == 0 {
			buffer_full = false
		}
	}

}

func main() { // Defines the main function, which is the entry point of the program
	listener, err := net.Listen("tcp", ":8080") // Listens for incoming TCP connections on port 8080 and returns a listener object and any error encountered
	if err != nil {                             // Checks if there was an error while listening.
		fmt.Println("Error listening: ", err) // Prints an error message if an error occurred
		return                                // Exits the function
	}
	defer listener.Close() // Defers the closing of the listener until the function returns

	fmt.Println("Server listening on port 8080") // Prints a message indicating that the server is listening on port 8080

	// Infinitely loop to accept incoming connections
	for {
		conn, err := listener.Accept() // Accepts the  incoming connection and returns a connection object and any error encountered
		if err != nil {                // Checks if there was an error while accepting the connection
			fmt.Println("Error accepting connection: ", err) // Prints an error message if an error occurred
			continue                                         // Skips the current iteration of the loop and proceeds to the next iteration
		}
		go handleConnection(conn) // Calls the handleConnection function in a new goroutine, allowing the server to handle multiple connections concurrently
	}
}
