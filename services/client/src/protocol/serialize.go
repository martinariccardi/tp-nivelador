package protocol

import (
	"bytes"
	"encoding/binary"
)

// SerializeTlvMessage encodes a TLV message with a 2-byte type and a 2-byte
// length (big-endian) followed by `payload` bytes.
func SerializeTlvMessage(msgType uint16, payload []byte) ([]byte, error) {
	message := new(bytes.Buffer)
	if err := binary.Write(message, binary.BigEndian, msgType); err != nil {
		return nil, err
	}
	if err := binary.Write(message, binary.BigEndian, uint16(len(payload))); err != nil {
		return nil, err
	}
	message.Write(payload)
	return message.Bytes(), nil
}

// SerializeEndMessage creates a TLV_END_TYPE message whose payload is the
// ASCII agency identifier. The server uses this message to detect the end
// of the client's transmission.
func SerializeEndMessage(agencyId string) ([]byte, error) {
	payload := []byte(agencyId)
	return SerializeTlvMessage(TLV_END_TYPE, payload)
}

// SerializeBet encodes a single Bet into an inner TLV sequence that contains
// the individual bet fields as TLV entries. The returned bytes are then
// wrapped into a TLV_BET_TYPE message by the caller.
func SerializeBet(bet Bet) ([]byte, error) {
	fields := []string{
		bet.AgencyId,
		bet.FirstName,
		bet.LastName,
		bet.Id,
		bet.Birthdate,
		bet.BetNumber,
	}

	payload := new(bytes.Buffer)
	// Build body
	for i, field := range fields {
		// Type
		if err := binary.Write(payload, binary.BigEndian, uint16(i+1)); err != nil {
			return nil, err
		}
		// Size
		if err := binary.Write(payload, binary.BigEndian, uint16(len(field))); err != nil {
			return nil, err
		}
		// Value
		payload.WriteString(field)
	}

	return SerializeTlvMessage(TLV_BET_TYPE, payload.Bytes())
}

// SerializeBatch concatenates the serialized representation of multiple
// bets and wraps them into a TLV_NEW_BATCH_TYPE message suitable for
// transmission to the server.
func SerializeBatch(bets []Bet) ([]byte, error) {
	payload := new(bytes.Buffer)
	for _, bet := range bets {
		serializedBet, err := SerializeBet(bet)
		if err != nil {
			return nil, err
		}
		_, err = payload.Write(serializedBet)
		if err != nil {
			return nil, err
		}
	}
	return SerializeTlvMessage(TLV_NEW_BATCH_TYPE, payload.Bytes())
}
