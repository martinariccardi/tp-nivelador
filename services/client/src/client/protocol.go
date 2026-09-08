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
const TLV_NEW_BATCH_TYPE uint16 = 0x04
const TLV_ACK_TYPE uint16 = 0x05
const TLV_NACK_TYPE uint16 = 0x06
const EXPECTED_FIELDS = 6
const TLV_HEADER_SIZE = 4

func serialize_tlv_message(msgType uint16, payload []byte) ([]byte, error) {
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

func serialize_batch(bets []Bet) ([]byte, error) {
	payload := new(bytes.Buffer)
	for _, bet := range bets {
		serializedBet, err := serialize_bet(bet)
		if err != nil {
			return nil, err
		}
		_, err = payload.Write(serializedBet)
		if err != nil {
			return nil, err
		}
	}
	return serialize_tlv_message(TLV_NEW_BATCH_TYPE, payload.Bytes())
}

func deserialize_ack(socket io.Reader) error {
	header, err := safe_socket.RecvAll(socket, TLV_HEADER_SIZE)
	if err != nil {
		return err
	}

	tlvType := binary.BigEndian.Uint16(header[0:2])
	tlvSize := int(binary.BigEndian.Uint16(header[2:4]))

	if tlvSize > 0 {
		return fmt.Errorf("formato de mensaje incorrecto")
	}

	switch tlvType {
	case TLV_ACK_TYPE:
		return nil
	case TLV_NACK_TYPE:
		return fmt.Errorf("el servidor rechazó el batch")
	default:
		return fmt.Errorf("tipo de mensaje inesperado")
	}
}

func deserialize_winners(socket io.Reader) ([]Bet, error) {
	header, err := safe_socket.RecvAll(socket, TLV_HEADER_SIZE)
	if err != nil {
		return []Bet{}, err
	}

	tlvType := binary.BigEndian.Uint16(header[0:2])
	tlvSize := int(binary.BigEndian.Uint16(header[2:4]))

	switch tlvType {
	case TLV_WINNER_TYPE:
		tlvValue, err := safe_socket.RecvAll(socket, tlvSize)
		if err != nil {
			return []Bet{}, err
		}
		return extractBets(tlvValue)
	default:
		return []Bet{}, fmt.Errorf("tipo de mensaje inesperado")
	}
}

func extractBet(rawBet []byte) (Bet, error) {
	index := 0
	var elems []string

	for index < len(rawBet) {
		if index+TLV_HEADER_SIZE > len(rawBet) {
			return Bet{}, fmt.Errorf("formato de apuesta incorrecto")
		}
		length := int(binary.BigEndian.Uint16(rawBet[index+2 : index+4]))
		index += TLV_HEADER_SIZE
		if index+length > len(rawBet) {
			return Bet{}, fmt.Errorf("formato de apuesta incorrecto")
		}
		content := string(rawBet[index : index+length])
		elems = append(elems, content)
		index += length
	}

	if len(elems) != EXPECTED_FIELDS {
		return Bet{}, fmt.Errorf("formato de apuesta incorrecto")
	}

	return Bet{
		AgencyId:  elems[0],
		FirstName: elems[1],
		LastName:  elems[2],
		Id:        elems[3],
		Birthdate: elems[4],
		BetNumber: elems[5],
	}, nil
}

func extractBets(payload []byte) ([]Bet, error) {
	index := 0
	var bets []Bet

	for index < len(payload) {
		if index+TLV_HEADER_SIZE > len(payload) {
			return []Bet{}, fmt.Errorf("formato de apuesta incorrecto")
		}
		length := int(binary.BigEndian.Uint16(payload[index+2 : index+4]))
		index += TLV_HEADER_SIZE
		if index+length > len(payload) {
			return []Bet{}, fmt.Errorf("formato de apuesta incorrecto")
		}
		content := payload[index : index+length]
		bet, err := extractBet(content)
		if err != nil {
			return []Bet{}, err
		}
		bets = append(bets, bet)
		index += length
	}

	return bets, nil
}
