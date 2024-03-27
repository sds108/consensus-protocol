// Boot up sequence for a server type node

// Version 1.3

package main

func main() {
	// Check if this server already has a Conversation ID

	// Generate a Conversation ID if necessary

	// Make Conversations List, with Mutex Lock
	conversations_lock.Lock()
	conversations = make(map[uint32]*conversation)
	conversations_lock.Unlock()

	// Start Listener Thread
	go listener(SERVER_PORT_CONST)
}
