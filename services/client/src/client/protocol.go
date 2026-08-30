package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const TLV_BET_TYPE uint16 = 0x01
const TLV_END_TYPE uint16 = 0x02
const TLV_WINNER_TYPE uint16 = 0x03
const EXPECTED_FIELDS = 6
const TLV_HEADER_SIZE = 4

func serialize_tlv_message(msg_type uint16, payload []byte) ([]byte, error) {
	message := new(bytes.Buffer)
	if err := binary.Write(message, binary.BigEndian, msg_type); err != nil {
		return nil, err
	}
	if err := binary.Write(message, binary.BigEndian, uint16(len(payload))); err != nil {
		return nil, err
	}
	message.Write(payload)
	return message.Bytes(), nil
}

func serialize_end_message(agencyId string) ([]byte, error) {
	payload := []byte(agencyId)
	return serialize_tlv_message(TLV_END_TYPE, payload)
}

func serialize_bet(bet Bet) ([]byte, error) {
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

	return serialize_tlv_message(TLV_BET_TYPE, payload.Bytes())
}

// Modularizar
func deserialize(socket io.Reader) ([]Bet, error) {
	header, err := safe_socket.RecvAll(socket, TLV_HEADER_SIZE)
	if err != nil {
		return nil, err
	}

	tlv_type := binary.BigEndian.Uint16(header[0:2])
	tlv_size := int(binary.BigEndian.Uint16(header[2:4]))

	if tlv_type != TLV_WINNER_TYPE {
		return nil, fmt.Errorf("tipo de mensaje inesperado")
	}

	tlv_value, err := safe_socket.RecvAll(socket, tlv_size)
	if err != nil {
		return nil, err
	}

	index := 0
	params := 0
	var currentBet Bet
	var winners []Bet
	for index < tlv_size {
		tlvType := binary.BigEndian.Uint16(tlv_value[index : index+2])
		length := int(binary.BigEndian.Uint16(tlv_value[index+2 : index+4]))
		index += TLV_HEADER_SIZE

		content := string(tlv_value[index : index+length])

		switch tlvType {
		case 1, 0x0011:
			currentBet.AgencyId = content
			params++
		case 2, 0x0012:
			currentBet.FirstName = content
			params++
		case 3, 0x0013:
			currentBet.LastName = content
			params++
		case 4, 0x0014:
			currentBet.Id = content
			params++
		case 5, 0x0015:
			currentBet.Birthdate = content
			params++
		case 6, 0x0016:
			currentBet.BetNumber = content
			params++
		}

		index += length

		if params == EXPECTED_FIELDS {
			winners = append(winners, currentBet)
			currentBet = Bet{}
			params = 0
		}
	}

	return winners, nil
}
