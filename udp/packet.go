// packet.go
// Defines the packet structure and provides functions for packet serialization and deserialization
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	//"fmt"
)

// Low Level Packet, with reliable data transfer features for UDP
const magic = 0x01051117

// Total Header size is 20 Bytes
type PcktHeader struct {
	Magic       uint32 // 4 bytes
	Checksum    uint32 // 4 bytes CRC32 (There is a hash/crc32 package for this, but not sure if we are allowed to use or what the approach is to this?)
	ConvID      uint32 // 4 bytes
	SequenceNum uint32 // 4 bytes
	IsFinal     uint16 // 2 bytes
	Type        uint16 // 2 bytes
}

// Types
const (
	Data = 0
	// ACK  = 1
	// NACK = 2
)

type Pckt struct {
	Header         PcktHeader // 20 bytes
	Body           []byte     // N bytes
	MissingPackets []uint32

	// 20 + N <= 256 Bytes
}

// SerializePacket serializes the entire packet including its header and body.
func SerializePacket(packet Pckt) ([]byte, error) {
	headerBytes, err := SerializeHeader(packet.Header)
	if err != nil {
		return nil, err
	}
	return append(headerBytes, packet.Body...), nil
}

// DeserializePacket deserializes the entire packet from bytes.
func DeserializePacket(data []byte) (*Pckt, error) {
	if len(data) < 20 { // Assuming header is 20 bytes
		return nil, errors.New("packet too short")
	}
	header, err := DeserializeHeader(data[:20])
	if err != nil {
		return nil, err
	}
	return &Pckt{Header: *header, Body: data[20:]}, nil
}

// SerializeHeader serializes the packet header into bytes.
func SerializeHeader(pcktheader PcktHeader) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, pcktheader.Magic); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, pcktheader.Checksum); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, pcktheader.ConvID); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, pcktheader.SequenceNum); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, pcktheader.IsFinal); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, pcktheader.Type); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DeserializeHeader deserializes the packet header from bytes.
func DeserializeHeader(data []byte) (*PcktHeader, error) {
	var header PcktHeader
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.BigEndian, &header.Magic); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &header.Checksum); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &header.ConvID); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &header.SequenceNum); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &header.IsFinal); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &header.Type); err != nil {
		return nil, err
	}

	return &header, nil
}
