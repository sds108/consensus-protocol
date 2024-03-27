// packet.go
// Defines the packet structure and provides functions for packet serialization and deserialization
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

// Low Level Packet, with reliable data transfer features for UDP

// Total Header size is 20 Bytes
type PcktHeader struct {
	Magic       uint32 // 4 bytes, used for critical situations
	Checksum    uint32 // 4 bytes CRC32 IEEE
	ConvID      uint32 // 4 bytes
	SequenceNum uint32 // 4 bytes
	IsFinal     uint16 // 2 bytes
	Type        uint16 // 2 bytes
}

// Types
const (
	Data = uint16(0)
	ACK  = uint16(1)
	NAK  = uint16(2)
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

// Returns true or false based on if byte slice a == b
func ByteSliceEqual(a []byte, b []byte) bool {
	// If not equal in length, obviously not equal
	if len(a) != len(b) {
		return false
	}

	// Check each index in slice
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	// if passed all tests, return true
	return true
}

// Used both to generate checksums for outgoing packets, and verify checksums for incoming packets
func ComputeChecksum(raw_packet []byte) ([]byte, error) {
	// Check raw_packet is the right size
	if len(raw_packet) < 20 {
		return nil, errors.New("ComputeChecksum: Too few bytes in argument: raw_packet")
	}

	// Compute CRC32 IEEE Checksum from ConvID to Type Fields
	CHECKSUM := crc32.ChecksumIEEE(raw_packet[8:20])

	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, CHECKSUM)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("ComputeChecksum: Error converting CRC32 Checksum to bytes")
	}

	return buf.Bytes(), nil
}

// Used to verify checkums for incoming packets, returns true if all checks out
func VerifyChecksum(raw_packet []byte) (bool, error) {
	// Check raw_packet is the right size
	if len(raw_packet) < 20 {
		return false, errors.New("VerifyChecksum: Too few bytes in argument: raw_packet")
	}

	CHECKSUM_BYTES, err := ComputeChecksum(raw_packet)
	if err != nil {
		fmt.Println(err)
		return false, errors.New("VerifyChecksum: ComputeChecksum returned Error")
	}

	return ByteSliceEqual(CHECKSUM_BYTES, raw_packet[4:8]), nil
}

// Used to verify the Magic field for incoming packets, returns true if all checks out
func VerifyMagic(raw_packet []byte) (bool, error) {
	// Check raw_packet is the right size
	if len(raw_packet) < 20 {
		return false, errors.New("VerifyMagic: Too few bytes in argument: raw_packet")
	}

	return ByteSliceEqual(MAGIC_BYTES_CONST(), raw_packet[0:4]), nil
}

// Used to verify checksums and the Magic field for incoming packets, returns true if all checks out
func VerifyPacket(raw_packet []byte) (bool, error) {
	// Check raw_packet is the right size
	if len(raw_packet) < 20 {
		return false, errors.New("VerifyPacket: Too few bytes in argument: raw_packet")
	}

	// Make sure Magic checks out
	verified_magic, err := VerifyMagic(raw_packet)
	if err != nil {
		return false, errors.New("VerifyPacket: VerifyMagic returned Error")
	}

	// Make sure the Checksum is correct
	verified_checksum, err := VerifyChecksum(raw_packet)
	if err != nil {
		return false, errors.New("VerifyPacket: VerifyChecksum returned Error")
	}

	// If both are true, return true, else return false
	return verified_magic && verified_checksum, nil
}
