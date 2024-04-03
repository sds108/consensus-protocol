// Packet inside Packet, Communication and Consensus Packet Structures

// Version 1.3
package main

import (
	"bytes"
	"encoding/binary"

	"github.com/google/uuid"
)

// /// Hello Packet
type PcktHello struct {
	DataID      uint16   // 2 bytes
	Version     uint32   // 4 bytes
	NumFeatures uint16   // 2 bytes
	Features    []uint16 // num_features*2 bytes
}

type PcktHelloResponse PcktHello

////// End of Hello

// /// Vote begin request
type PcktVoteRequest struct {
	DataID         uint16    // 2 bytes
	VoteID         uuid.UUID // 16 bytes
	QuestionLength uint32    // 4 bytes
	Question       string    // QuestionLength bytes long
}

// /// Vote Broadcast,
type PcktVoteBroadcast PcktVoteRequest

// /// Client Response, Server will also time and TIMEOUT waiting for a response if necessary
type PcktVoteResponse struct {
	DataID   uint16    // 2 bytes
	VoteID   uuid.UUID // 16 bytes
	Response uint16    // 2 bytes
}

// /// Result Broadcast, once the server has received a satisfactory amount of votes, it calculates the winner and broadcasts the winning result
type PcktVoteResultBroadcast PcktVoteResponse

// Used to check which Deserializer to use
func DeserializeDataID(raw_data []byte) (uint16, error) {
	var DataID uint16

	buf := bytes.NewReader(raw_data)

	if err := binary.Read(buf, binary.BigEndian, &DataID); err != nil {
		return 0, err
	} else {
		return DataID, nil
	}
}

// Deserialize Hello or Hello Response Packet
func DeserializeHello(raw_data []byte) (*PcktHello, error) {
	var pckthello PcktHello

	buf := bytes.NewReader(raw_data)

	if err := binary.Read(buf, binary.BigEndian, &pckthello.DataID); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.BigEndian, &pckthello.Version); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.BigEndian, &pckthello.NumFeatures); err != nil {
		return nil, err
	}

	// Create a slice the size of Num of Features
	pckthello.Features = make([]uint16, pckthello.NumFeatures)

	if err := binary.Read(buf, binary.BigEndian, &pckthello.Features); err != nil {
		return nil, err
	}

	return &pckthello, nil
}

// Serialize Hello or Hello Response Packet
func SerializeHello(pckthello *PcktHello) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Store Data ID
	if err := binary.Write(buf, binary.BigEndian, pckthello.DataID); err != nil {
		return nil, err
	}

	// Store Version
	if err := binary.Write(buf, binary.BigEndian, pckthello.Version); err != nil {
		return nil, err
	}

	// Store Actual Length of the Features Slice
	if err := binary.Write(buf, binary.BigEndian, uint16(len(pckthello.Features))); err != nil {
		return nil, err
	}

	// Store the Features
	if err := binary.Write(buf, binary.BigEndian, pckthello.Features); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Deserialize Vote Request or Broadcast Packet
func DeserializeVoteRequest(raw_data []byte) (*PcktVoteRequest, error) {
	var pcktvoterequest PcktVoteRequest

	buf := bytes.NewReader(raw_data)

	// Extract the Data ID
	if err := binary.Read(buf, binary.BigEndian, &pcktvoterequest.DataID); err != nil {
		return nil, err
	}

	// Extract the Vote ID
	if err := binary.Read(buf, binary.BigEndian, &pcktvoterequest.VoteID); err != nil {
		return nil, err
	}

	// Extract the Question Length to create an appropriate string buffer for the Question
	if err := binary.Read(buf, binary.BigEndian, &pcktvoterequest.QuestionLength); err != nil {
		return nil, err
	}

	// Create a byte slice to hold the Question String
	question_bytes := make([]byte, pcktvoterequest.QuestionLength)

	// Extract the Question String
	if err := binary.Read(buf, binary.BigEndian, question_bytes); err != nil {
		return nil, err
	}

	// Convert question_bytes to a string and store to the Question attribute in the struct
	pcktvoterequest.Question = string(question_bytes)

	return &pcktvoterequest, nil
}

// Serialize Vote Request or Broadcast Packet
func SerializeVoteRequest(pcktvoterequest *PcktVoteRequest) ([]byte, error) {

	buf := new(bytes.Buffer)

	// Store the Data ID
	if err := binary.Write(buf, binary.BigEndian, pcktvoterequest.DataID); err != nil {
		return nil, err
	}

	// Store the Vote ID
	if err := binary.Write(buf, binary.BigEndian, pcktvoterequest.VoteID); err != nil {
		return nil, err
	}

	// Store the actual question length by checking the Question string again
	if err := binary.Write(buf, binary.BigEndian, uint32(len(pcktvoterequest.Question))); err != nil {
		return nil, err
	}

	// Store the Question String
	if err := binary.Write(buf, binary.BigEndian, []byte(pcktvoterequest.Question)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Deserialize Vote Response or Broadcast Result Packet
func DeserializeVoteResponse(raw_data []byte) (*PcktVoteResponse, error) {
	var pcktvoteresponse PcktVoteResponse

	buf := bytes.NewReader(raw_data)

	// Extract the Data ID
	if err := binary.Read(buf, binary.BigEndian, &pcktvoteresponse.DataID); err != nil {
		return nil, err
	}

	// Extract the Vote ID
	if err := binary.Read(buf, binary.BigEndian, &pcktvoteresponse.VoteID); err != nil {
		return nil, err
	}

	// Extract the Vote Response
	if err := binary.Read(buf, binary.BigEndian, &pcktvoteresponse.Response); err != nil {
		return nil, err
	}

	return &pcktvoteresponse, nil
}

// Serialize Vote Response or Broadcast Result Packet
func SerializeVoteResponse(pcktvoteresponse *PcktVoteResponse) ([]byte, error) {

	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, pcktvoteresponse.DataID); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, pcktvoteresponse.VoteID); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, pcktvoteresponse.Response); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
