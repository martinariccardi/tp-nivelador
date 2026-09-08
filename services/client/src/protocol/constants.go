package protocol

const (
	TLV_BET_TYPE       uint16 = 0x01
	TLV_END_TYPE       uint16 = 0x02
	TLV_WINNER_TYPE    uint16 = 0x03
	TLV_NEW_BATCH_TYPE uint16 = 0x04
	TLV_ACK_TYPE       uint16 = 0x05
	TLV_NACK_TYPE      uint16 = 0x06
	EXPECTED_FIELDS           = 6
	TLV_HEADER_SIZE           = 4
)
