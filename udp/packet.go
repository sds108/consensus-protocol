// Packet handling for packet serialisation and deserialisation

// Version 1.3
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Low Level Packet, with reliable data transfer features for UDP
const magic = 0x01051117

// Total Header size is 20 Bytes
type PcktHeader struct {
	Magic       uint32 // 4 bytes
	Checksum    uint32 // 4 bytes CRC32
	ConvID      uint32 // 4 bytes
	SequenceNum uint32 // 4 bytes
	Final       uint16 // 2 bytes
	Type        uint16 // 2 bytes
}

// Types
const (
	Data = 0
	ACK  = 1
	NACK = 2
)

type Pckt struct {
	Header PcktHeader // 20 bytes
	Body   []byte     // N bytes

	// 20 + N <= 256 Bytes
}

// Deserialise Header
func DeserializeHeader(raw_header []byte) (*PcktHeader, error) {
	// New Packet Header
	var MAGIC uint32
	var CHECKSUM uint32
	var CONVID uint32
	var SEQUENCENUM uint32
	var FINAL uint16
	var TYPE uint16

	err := binary.Read(bytes.NewReader(raw_header[0:4]), binary.BigEndian, &MAGIC)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Magic Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[4:8]), binary.BigEndian, &CHECKSUM)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Checksum Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[8:12]), binary.BigEndian, &CONVID)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the ConvID Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[12:16]), binary.BigEndian, &SEQUENCENUM)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the SequenceNum Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[16:18]), binary.BigEndian, &FINAL)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Final Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[18:20]), binary.BigEndian, &TYPE)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Type Field")
	}

	return &PcktHeader{
		Magic:       MAGIC,
		Checksum:    CHECKSUM,
		ConvID:      CONVID,
		SequenceNum: SEQUENCENUM,
		Final:       FINAL,
		Type:        TYPE,
	}, nil
}

// Serialize Header
func SerializeHeader(pcktheader PcktHeader) ([]byte, error) {

	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Magic Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Checksum Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the ConvID Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the SequenceNum Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Final Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Type Field")
	}

	return buf.Bytes(), nil
}

/*
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
*/
