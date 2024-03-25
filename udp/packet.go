// Packet handling for packet serialisation and deserialisation

// Version 1.3
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
	Final       uint16 // 2 bytes
	Type        uint16 // 2 bytes
}

// Types
const (
	Data = uint16(0)
	ACK  = uint16(1)
	NACK = uint16(2)
)

type Pckt struct {
	Header PcktHeader // 20 bytes
	Body   []byte     // N bytes

	// 20 + N <= 256 Bytes
}

// Deserialise Header
func DeserializeHeader(raw_header []byte) (*PcktHeader, error) {
	// New Packet Header

	// Temporary variables
	var MAGIC uint32
	var CHECKSUM uint32
	var CONVID uint32
	var SEQUENCENUM uint32
	var FINAL uint16
	var TYPE uint16

	err := binary.Read(bytes.NewReader(raw_header[0:4]), binary.BigEndian, &MAGIC)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("DeserializeHeader: Error decoding the Magic Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[4:8]), binary.BigEndian, &CHECKSUM)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("DeserializeHeader: Error decoding the Checksum Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[8:12]), binary.BigEndian, &CONVID)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("DeserializeHeader: Error decoding the ConvID Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[12:16]), binary.BigEndian, &SEQUENCENUM)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("DeserializeHeader: Error decoding the SequenceNum Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[16:18]), binary.BigEndian, &FINAL)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("DeserializeHeader: Error decoding the Final Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[18:20]), binary.BigEndian, &TYPE)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("DeserializeHeader: Error decoding the Type Field")
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
		return nil, errors.New("SerializeHeader: Error encoding the Magic Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("SerializeHeader: Error encoding the Checksum Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("SerializeHeader: Error encoding the ConvID Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("SerializeHeader: Error encoding the SequenceNum Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("SerializeHeader: Error extracting the Final Field")
	}

	err = binary.Write(buf, binary.BigEndian, pcktheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("SerializeHeader: Error extracting the Type Field")
	}

	return buf.Bytes(), nil
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
