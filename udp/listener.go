package main

import (
	//"fmt"

	"log"
	"net"
)

func listener() {
	// Listen Continuously
	for true {
		buffer := make([]byte, 8192)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error reading from UDP:", err)
			continue
		}

		go handleIncomingPackets(conn, addr, buffer[:n])
	}
}

func handleIncomingPackets(conn *net.UDPConn, addr *net.UDPAddr, raw_packet []byte) {

	// Make sure Data is at least 24 Bytes
	if len(raw_packet) < 24 {
		log.Printf("handleIncomingPackets: insufficient packet size\n")
		return
	}

	// Magic and Checksum check, if this fails, you would drop the packet
	verify_packet, err := VerifyPacket(raw_packet)
	if err != nil {
		log.Printf("handleIncomingPackets: VerifyPacket returned Error\n")
		return
	}

	if !verify_packet {
		log.Printf("handleIncomingPackets: VerifyPacket returned False\n")
		return
	}

	// Deserialize the Header
	packet, err := DeserializePacket(raw_packet)
	if err != nil {
		log.Printf("handleIncomingPackets: DeserializeHeader returned Error\n")
		return
	}

	// Check if Ping Request for Conversation ID Assignment
	if packet.Header.Type == PING_REQ {
		pingPckt := Pckt{
			Header: PcktHeader{
				Magic:       MAGIC_CONST,
				Checksum:    0,
				ConvID:      generateConversationID(),
				PacketNum:   0,
				SequenceNum: 0,
				Type:        PING_RES,
				IsFinal:     1,
			},
			Body: []byte{},
		}

		// Send Back Unique Conversation ID for the Client
		sendUDP(addr, &pingPckt)
		return
	}

	// Check Got Assigned a new Conversation ID
	if packet.Header.Type == PING_RES {
		if conversation_id_self == 0 {
			conversation_id_self = packet.Header.ConvID
		}

		return // Drop packet, to not accidentally create a conversation with yourself
	}

	// Block incoming if haven't got a Conversation ID
	if conversation_id_self == 0 {
		return
	}

	conversations_lock.Lock()

	conversationRef, exists := conversations[packet.Header.ConvID]
	if !exists {
		conversationRef = newConversation(packet.Header.ConvID, addr)
		conversations[packet.Header.ConvID] = conversationRef
		conversationRef.startConversation()
	}
	conversations_lock.Unlock()

	conversationRef.ARQ_Receive(conn, addr, *packet)
}
