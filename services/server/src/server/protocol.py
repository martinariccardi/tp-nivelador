import safe_socket
from lottery.bet import Bet

TLV_BET_TYPE = 0x01
TLV_END_TYPE = 0x02
TLV_WINNERS_TYPE = 0x03
EXPECTED_FIELDS = 6
TLV_HEADER_SIZE = 4

def serialize_tlv_message(msg_type, payload):
    header = msg_type.to_bytes(2, byteorder='big') + len(payload).to_bytes(2, byteorder='big')
    return header + bytes(payload)

def serialize_bet(bet):
    fields = [
        str(bet.agency_id),
        str(bet.first_name),
        str(bet.last_name),
        str(bet.document),
        str(bet.birthdate),
        str(bet.number),
    ]
    
    payload = bytearray()
    for field_index, field in enumerate(fields, start=1):
        field_bytes = field.encode("utf-8")
        payload.extend(field_index.to_bytes(2, byteorder='big'))
        payload.extend(len(field_bytes).to_bytes(2, byteorder='big'))
        payload.extend(field_bytes)

    return serialize_tlv_message(TLV_BET_TYPE, payload)


def serialize_winners(winners):
    payload = bytearray()
    for winner in winners:
        bet_bytes = serialize_bet(winner)
        # Ignore NEW_BET header
        payload.extend(bet_bytes[TLV_HEADER_SIZE:])
    return serialize_tlv_message(TLV_WINNERS_TYPE, payload)


def deserialize(socket):
    header = safe_socket.recv_all(socket, TLV_HEADER_SIZE)
    if not header:
        return None

    tlv_type = int.from_bytes(header[0:2], byteorder='big')
    tlv_size = int.from_bytes(header[2:4], byteorder='big')

    if tlv_type == TLV_BET_TYPE:
        tlv_value = safe_socket.recv_all(socket, tlv_size)
        return {
            "type": "NEW_BET",
            "data": extract_bet(tlv_value),
        }
    elif tlv_type == TLV_END_TYPE:
        tlv_value = safe_socket.recv_all(socket, tlv_size)
        agency_id = tlv_value.decode('utf-8')
        return {
            "type": "END_BETS",
            "data": agency_id,
        }
    elif tlv_type == TLV_WINNERS_TYPE:
        # manejar error
        pass
    else:
        raise ValueError(f"Tipo de mensaje no reconocido: {tlv_type}")


def extract_bet(bet):
    index = 0
    elems = []
    while index < len(bet):
        length = int.from_bytes(bet[index+2:index+4], byteorder='big')
        index += TLV_HEADER_SIZE
        content = bet[index:index + length].decode("utf-8")
        elems.append(content)
        index += length

    return Bet(
        agency_id=int(elems[0]),
        first_name=elems[1],
        last_name=elems[2],
        document=int(elems[3]),
        birthdate=elems[4],
        number=int(elems[5]),
    )


