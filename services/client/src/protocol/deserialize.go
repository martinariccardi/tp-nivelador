package protocol

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

func DeserializeAck(socket io.Reader) error {
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

func DeserializeWinners(socket io.Reader) ([]Bet, error) {
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
