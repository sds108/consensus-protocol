// Version 1.3
package main

import (
	"log"
	"math/rand"
	"net"
	"sync"
)

// This file includes all of the global variables necessary for the function of the Node

// Global Constants ////////////////////////////////////////////////////////////////////

// Types
const (
	DATA     uint16 = 0
	ACK      uint16 = 1
	NAK      uint16 = 2
	SYN      uint16 = 3
	SYN_ACK  uint16 = 4
	RESET    uint16 = 5
	PING_REQ uint16 = 0xFFFE
	PING_RES uint16 = 0xFFFF
)

// Responses
const (
	UNSAT        uint16 = 0
	SAT          uint16 = 1
	SYNTAX_ERROR uint16 = 2
	TIMEOUT      uint16 = 3
)

// Data IDs
const (
	hello_c2s                     uint16 = 0 // from client to server, basically SYN
	hello_back_s2c                uint16 = 1 // from server to client, basically ACK
	vote_c2s_request_vote         uint16 = 2 // from client to server to begin vote
	vote_s2c_broadcast_question   uint16 = 3 // from server to all clients to vote on
	vote_c2s_response_to_question uint16 = 4 // from client to server
	vote_s2c_broadcast_result     uint16 = 5 // from server to all clients
)

// Features
const (
	none        uint16 = 0
	simple_eval uint16 = 1
)

const MAGIC_CONST = 0x01051117

const MAX_PCKT_SIZE = 250

// Returns MAGIC_CONST in Big Endian Bytes
func MAGIC_BYTES_CONST() []byte {
	return []byte{byte(1), byte(5), byte(17), byte(23)}
}

const SERVER_PORT_CONST = "8080"

// Global Objects
var (
	i_am_server          bool
	debug_mode           bool
	loss_constant        float64
	serverAddr           *net.UDPAddr
	conn                 *net.UDPConn
	conversation_id_self uint32 = 0
	conversations_lock   sync.Mutex
	conversations        map[uint32]*conversation = make(map[uint32]*conversation)
	generatedConvIDs     map[uint32]bool          = make(map[uint32]bool)
	globalWaitGroup      sync.WaitGroup
	ref_manager          *referendum_manager = newReferendumManager()
	my_features          []uint16
)

func generateConversationID() uint32 {
	var newConversationID uint32 = 0

	_, exists := generatedConvIDs[newConversationID]

	// keep generating until unique
	for newConversationID == 0 || exists {
		newConversationID = rand.Uint32()
		_, exists = generatedConvIDs[newConversationID]
	}

	// Save into history bank
	generatedConvIDs[newConversationID] = true

	return newConversationID
}

// Global Functions
func sendUDP(addr *net.UDPAddr, pckt *Pckt) error {
	// Serialize Packet
	pckt_bytes, err := SerializePacket(pckt)
	if err != nil {
		if debug_mode {
			log.Printf("Error Serializing Packet %d: %d", pckt.Header.PacketNum, pckt.Header.SequenceNum)
		}
		return err
	}

	// Generate Checksum
	checksum_bytes, err := ComputeChecksum(pckt_bytes)
	if err != nil {
		if debug_mode {
			log.Printf("Error generating checksum for Packet %d: %d", pckt.Header.PacketNum, pckt.Header.SequenceNum)
		}
		return err
	}

	// Insert Checksum into the checksum field of the packet_bytes
	copy(pckt_bytes[4:8], checksum_bytes[:4])

	// Send it off over UDP
	if i_am_server {
		if _, err := conn.WriteTo(pckt_bytes, addr); err != nil {
			if debug_mode {
				log.Printf("Error sending packet %d: %d", pckt.Header.PacketNum, pckt.Header.SequenceNum)
			}
			return err
		}
	} else {
		if _, err := conn.Write(pckt_bytes); err != nil {
			if debug_mode {
				log.Printf("Error sending packet %d: %d", pckt.Header.PacketNum, pckt.Header.SequenceNum)
			}
			return err
		}
	}

	return nil
}
