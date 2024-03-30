// Boot up sequence for a client type node

// Version 1.3

package main

func main() {
	// Check if this client already has a Conversation ID

	// Ping server for a Conversation ID if necessary

	// Make Conversations List, with Mutex Lock
	conversations_lock.Lock()
	conversations = make(map[uint32]*conversation)
	conversations_lock.Unlock()

	// Start Listener Thread
	go listener(CLIENT_PORT_CONST)
}
