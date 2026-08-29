package client

import (
	"bytes"
	"io"
	"strings"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const TLV_BET_TYPE byte = 0x03
const EXPECTED_FIELDS = 5
const TLV_HEADER_SIZE = 2

func serialize(msg string) []byte {
	var serialized_msg bytes.Buffer
	parts := strings.Split(msg, ",")

	// Build header
	serialized_msg.WriteByte(TLV_BET_TYPE)
	serialized_msg.WriteByte(byte(len(msg) - EXPECTED_FIELDS - 1))
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

func deserialize(msg []byte, socket io.Reader) string {
	var deserialized_msg strings.Builder

	header, _ := safe_socket.RecvAll(socket, TLV_HEADER_SIZE)
	tlv_size := int(header[1])
	total_size := tlv_size + (EXPECTED_FIELDS * TLV_HEADER_SIZE)
	tlv_value, _ := safe_socket.RecvAll(socket, total_size)

	index := 0
	params := 0
	for index < total_size {
		lenght := int(tlv_value[index+1])
		index += TLV_HEADER_SIZE

		content := string(tlv_value[index : index+lenght])
		deserialized_msg.WriteString(content)

		if params < EXPECTED_FIELDS {
			deserialized_msg.WriteString(",")
		}

		index += lenght
	}

	return deserialized_msg.String()

}
