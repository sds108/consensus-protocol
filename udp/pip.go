// Packet inside Packet, Communication and Consensus Packet Structures

// Version 1.3
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const magic = 0x01051117

type DataHeader struct {
	Magic    uint32 // 4 bytes
	Checksum uint32 // 4 bytes CRC32
	DataID   uint16 // 2 bytes
}

// Data IDs
const (
	hello_c2s                     = 0 // from client to server, basically SYN
	hello_back_s2c                = 1 // from server to client, basically ACK
	vote_c2s_request_vote         = 2 // from client to server to begin vote
	vote_s2c_broadcast_question   = 3 // from server to all clients to vote on
	vote_c2s_response_to_question = 4 // from client to server
	vote_s2c_broadcast_result     = 5 // from server to all clients
)

///// Hello Packet

const (
	voting        = 0
	file_transfer = 1
)

type PcktHello struct {
	Header      DataHeader // 8 bytes
	Version     uint32     // 4 bytes
	NumFeatures uint16     // 2 bytes
	Feature     []uint16   // num_features*2 bytes
}

type PcktHelloResponse PcktHello

////// End of Hello

///// Vote begin request

type PcktVoteRequest struct {
	Header         DataHeader // 8 bytes
	VoteID         uuid.UUID  // 16 bytes // Golang: https://pkg.go.dev/github.com/beevik/guid Python: https://stackoverflow.com/questions/534839/how-to-create-a-guid-uuid-in-python,
	QuestionLength uint32     // 4 bytes
	Question       string     // QuestionLength bytes long, for Z3
}

// /// Vote Broadcast,
type PcktVoteBroadcast PcktVoteRequest

// Responses
const (
	UNSAT        = 0
	SAT          = 1
	SYNTAX_ERROR = 2
	TIMEOUT      = 3
)

// /// Client Response, Server will also time and TIMEOUT waiting for a response if necessary
type PcktVoteResponse struct {
	Header   DataHeader // 8 bytes
	VoteID   uuid.UUID  // 16 bytes
	Response uint16     // 2 bytes
}

// /// Result Broadcast, once the server has received a satisfactory amount of votes, it calculates the winner and broadcasts the winning result
type PcktVoteResultBroadcast PcktVoteResponse

// Deserialise Data Header
func DeserializeDataHeader(raw_header []byte) (*DataHeader, error) {
	// New Data Header
	var MAGIC uint32
	var CHECKSUM uint32
	var DATAID uint16

	err := binary.Read(bytes.NewReader(raw_header[0:4]), binary.BigEndian, &MAGIC)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Data Magic Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[4:8]), binary.BigEndian, &CHECKSUM)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Data Checksum Field")
	}

	err = binary.Read(bytes.NewReader(raw_header[8:10]), binary.BigEndian, &DATAID)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the DataID Field")
	}

	return &DataHeader{
		Magic:    MAGIC,
		Checksum: CHECKSUM,
		DataID:   DATAID,
	}, nil
}

// Serialize Data Header
func SerializeDataHeader(dataheader DataHeader) ([]byte, error) {

	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, dataheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Data Magic Field")
	}

	err = binary.Write(buf, binary.BigEndian, dataheader.Magic)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Data Checksum Field")
	}

	err = binary.Write(buf, binary.BigEndian, dataheader.DataID)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the DataID Field")
	}

	return buf.Bytes(), nil
}

// Deserialise Hello Packet
func DeserializeHello(DATAHEADER DataHeader, raw_data []byte) (*PcktHello, error) {
	// New Hello Packet
	var MAGIC uint32
	var VERSION uint32
	var NUMFEATURES uint16
	var FEATURE []uint16
	var temp_for_extraction uint16

	err := binary.Read(bytes.NewReader(raw_data[0:4]), binary.BigEndian, &MAGIC)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Hello Magic Field")
	}

	err = binary.Read(bytes.NewReader(raw_data[4:8]), binary.BigEndian, &VERSION)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Hello Version Field")
	}

	err = binary.Read(bytes.NewReader(raw_data[8:10]), binary.BigEndian, &NUMFEATURES)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong extracting the Num of Features Field")
	}

	for i := uint16(0); i < NUMFEATURES; i++ {
		err = binary.Read(bytes.NewReader(raw_data[(10+i):(10+i+2)]), binary.BigEndian, &temp_for_extraction)
		if err != nil {
			fmt.Println(err)
			return nil, errors.New("Something wrong extracting the Features Field")
		}

		// Add Features to Slice
		FEATURE = append(FEATURE, temp_for_extraction)
	}

	return &PcktHello{
		Header:      DATAHEADER,
		Version:     VERSION,
		NumFeatures: NUMFEATURES,
		Feature:     FEATURE,
	}, nil
}

// Serialize Data Header
func SerializeHello(pckthello PcktHello) ([]byte, error) {

	buf := new(bytes.Buffer)

	// Serialize the Data Header
	dataheader, err := SerializeDataHeader(pckthello.Header)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Data Header for the Hello Packet")
	}

	// Store Serialized Data Header to buf
	_, err = buf.Write(dataheader)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong appending the Data Header to the Hello Packet serialization buffer")
	}

	// Store Version Attribute to buf
	err = binary.Write(buf, binary.BigEndian, pckthello.Version)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Hello Packet Version Field")
	}

	// Store NumFeatures to buf
	err = binary.Write(buf, binary.BigEndian, pckthello.NumFeatures)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Something wrong encoding the Hello Packet Num Features Field")
	}

	// Store all Features to buf
	for i := uint16(0); i < pckthello.NumFeatures; i++ {
		err = binary.Write(buf, binary.BigEndian, pckthello.Feature[i])
		if err != nil {
			fmt.Println(err)
			return nil, errors.New("Something wrong encoding the Features Field in the Hello Packet")
		}
	}

	return buf.Bytes(), nil
}
