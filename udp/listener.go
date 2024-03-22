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
	conversations := make(map[uint32]*conversation)

	// Listen Continuously
	for {
		buffer := make([]byte, 4096)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error reading from UDP:", err)
			continue
		}

		go handleIncomingPackets(conn, addr, buffer[:n], &conversations)
	}
}

func handleIncomingPackets(conn *net.UDPConn, addr *net.UDPAddr, data []byte, conversations *map[uint32]*conversation) {
	// Make sure Data is over 20 Bytes
	if len(data) < 20 {
		log.Printf("Packet size below 20 Bytes???\n")
		return
	}

	pcktheader, err := DeserializeHeader(data)
	if err != nil {
		log.Printf("Error deserializing packet: %v", err)
		return
	}

	//fmt.Printf("Received packet: ConversationId=%s, SequenceNumber=%d, Data=%s, IsFinal=%t\n", packet.ConversationId, packet.SequenceNumber, packet.Data, packet.IsFinal)

	// Create Packet
	pckt := Pckt{*pcktheader, data[20:]}

	// Find Corresponding Conversation
	conversation_reference, exists := (*conversations)[pcktheader.ConvID]

	// If it exists, send the packet to the corresponding conversation
	if exists {
		conversation_reference.ARQ_Receive(conn, addr, pckt)
	} else {
		// Else create a new conversation
		(*conversations)[pcktheader.ConvID] = newConversation(pcktheader.ConvID)

		// and now send it there
		(*conversations)[pcktheader.ConvID].ARQ_Receive(conn, addr, pckt)
	}

	// Print size of conversations list
	fmt.Printf("Conversations: %d\n", len(*conversations))
}

func main() {
	listener()
}
