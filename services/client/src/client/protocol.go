package client

import (
	"bytes"
	"io"
	"strings"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

type Bet struct {
	FirstName string
	LastName  string
	Id        string
	Birthdate string
	BetNumber string
}

const TLV_BET_TYPE byte = 0x01
const TLV_END_TYPE byte = 0x02
const TLV_WINNER_TYPE byte = 0x03
const EXPECTED_FIELDS = 5
const TLV_HEADER_SIZE = 2

func serialize_tlv_message(msg_type byte, payload []byte) []byte {
	var message bytes.Buffer
	message.WriteByte(msg_type)
	message.WriteByte(byte(len(payload)))
	message.Write(payload)
	return message.Bytes()
}

func serialize_end_message(agencyId string) []byte {
	payload := []byte(agencyId)
	return serialize_tlv_message(TLV_END_TYPE, payload)
}

func serialize(msg string, agencyId string) []byte {
	var payload bytes.Buffer
	parts := strings.Split(msg, ",")

	// Add agency id
	payload.WriteByte(1)
	payload.WriteByte(byte(len(agencyId)))
	payload.WriteString(agencyId)

	// Build body
	for i, part := range parts {
		// Type
		payload.WriteByte(byte(i + 2))
		// Size
		payload.WriteByte(byte(len(part)))
		// Value
		payload.WriteString(part)
	}
	return serialize_tlv_message(TLV_BET_TYPE, payload.Bytes())
}

func deserialize(socket io.Reader) ([]Bet, error) {
	header, err := safe_socket.RecvAll(socket, TLV_HEADER_SIZE)
	if err != nil {
		return nil, err
	}

	tlv_size := int(header[1])
	tlv_value, err := safe_socket.RecvAll(socket, tlv_size)
	if err != nil {
		return nil, err
	}

	index := 0
	params := 0
	var currentFields [EXPECTED_FIELDS]string
	var winners []Bet
	for index < tlv_size {
		tlvType := int(tlv_value[index])
		length := int(tlv_value[index+1])
		index += TLV_HEADER_SIZE

		content := string(tlv_value[index : index+length])

		if tlvType >= 1 && params < EXPECTED_FIELDS {
			currentFields[tlvType-1] = content
		}
		params++
		index += length

		if params == EXPECTED_FIELDS {
			bet := Bet{
				FirstName: currentFields[0],
				LastName:  currentFields[1],
				Id:        currentFields[2],
				Birthdate: currentFields[3],
				BetNumber: currentFields[4],
			}
			winners = append(winners, bet)
			params = 0
		}
	}

	return winners, nil

}
