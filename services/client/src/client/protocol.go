package client

import (
	"bytes"
	"strings"
)

const TLV_BET_TYPE byte = 0x03
const COMMA_AMOUNT = 4 // cambiar nombre

func serialize(msg string) []byte {
	var serialized_msg bytes.Buffer
	parts := strings.Split(msg, ",")

	// Build header
	serialized_msg.WriteByte(TLV_BET_TYPE)
	serialized_msg.WriteByte(byte(len(msg) - COMMA_AMOUNT))
	// Build body
	for i, part := range parts {
		// Type
		serialized_msg.WriteByte(byte(i + 1))
		// Size
		serialized_msg.WriteByte(byte(len(part)))
		// Value
		serialized_msg.WriteString(part)
	}

	return serialized_msg.Bytes()
}

func deserialize()
