// Packet handling for packet serialisation and deserialisation

// Version 1.3
package main

import (
	"encoding/json"
)

type Packet struct {
	ConversationId string `json:"conversation_id"`
	SequenceNumber int    `json:"sequence_number"`
	Data           string `json:"data"`
	IsFinal        bool   `json:"is_final"`
	Ack            bool   `json:"ack"`             // Indicates if the packet is an ACK
	MissingPackets []int  `json:"missing_packets"` // SACK: List of missing packets
}

func serializePacket(packet Packet) ([]byte, error) {
	return json.Marshal(packet)
}

func deserializePacket(data []byte) (Packet, error) {
	var packet Packet
	err := json.Unmarshal(data, &packet)
	return packet, err
}
