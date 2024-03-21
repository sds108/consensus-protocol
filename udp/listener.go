// Focuses on the server side logic.
// Includes listening for incoming packets and handlin them.

// Version 1.3

package main

import (
	"fmt"
	"log"
	"net"
)

var udpPort = "8080"

func listener() {
	// Resolve UDP Address to listen at
	addr, err := net.ResolveUDPAddr("udp", ":"+udpPort)
	if err != nil {
		log.Fatal(err)
	}

	// Start Connection Socket on the UDP Address
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()
	log.Printf("UDP server listening on port %s", udpPort)

	// Initialize a list of current conversations
	conversations := make(map[string]*conversation)

	// Listen Continuously
	for {
		buffer := make([]byte, 1024)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error reading from UDP:", err)
			continue
		}

		go handleIncomingPackets(conn, addr, buffer[:n], &conversations)
	}
}

func handleIncomingPackets(conn *net.UDPConn, addr *net.UDPAddr, data []byte, conversations *map[string]*conversation) {
	packet, err := deserializePacket(data)
	if err != nil {
		log.Printf("Error deserializing packet: %v", err)
		return
	}

	//fmt.Printf("Received packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", packet.ConversationId, packet.SequenceNumber, packet.Data, packet.IsFinal)

	// Find Corresponding Conversation
	conversation_reference, exists := (*conversations)[packet.ConversationId]

	// If it exists, send the packet to the corresponding conversation
	if exists {
		conversation_reference.ARQ_Receive(conn, addr, packet)
	} else {
		// Else create a new conversation
		(*conversations)[packet.ConversationId] = newConversation(packet.ConversationId)

		// and now send it there
		(*conversations)[packet.ConversationId].ARQ_Receive(conn, addr, packet)
	}

	// Print size of conversations list
	fmt.Printf("Conversations: %d\n", len(*conversations))
}

func main() {
	listener()
}
