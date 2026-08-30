import safe_socket
from lottery.bet import Bet

TLV_BET_TYPE = 0x01
TLV_END_TYPE = 0x02
TLV_WINNERS_TYPE = 0x03
EXPECTED_FIELDS = 5
TLV_HEADER_SIZE = 2

def serialize_tlv_message(msg_type, payload):
    return bytes([msg_type, len(payload)]) + bytes(payload)

def serialize_bet(bet):
    fields = [
        str(bet.first_name),
        str(bet.last_name),
        str(bet.document),
        str(bet.birthdate),
        str(bet.number),
    ]
    
    payload = bytearray()
    for field_index, field in enumerate(fields, start=1):
        field_bytes = field.encode("utf-8")
        payload.append(field_index)
        payload.append(len(field_bytes))
        payload.extend(field_bytes)

    return serialize_tlv_message(TLV_BET_TYPE, payload)


def serialize_winners(winners):
    payload = bytearray()
    for winner in winners:
        bet_bytes = serialize_bet(winner)
        # Ignore NEW_BET header
        payload.extend(bet_bytes[2:])
    return serialize_tlv_message(TLV_WINNERS_TYPE, payload)


def deserialize(socket):
    header = safe_socket.recv_all(socket, TLV_HEADER_SIZE)
    print("header:", header, list(header))
    if not header:
        return None

    tlv_type = header[0]
    tlv_size = header[1]

    if tlv_type == TLV_BET_TYPE:
        tlv_value = safe_socket.recv_all(socket, tlv_size)
        return {
            "type": "NEW_BET",
            "data": extract_bet(tlv_value),
        }
    elif tlv_type == TLV_END_TYPE:
        return {
            "type": "END_BETS",
            "data": None,
        }
    elif tlv_type == TLV_WINNERS_TYPE:
        pass
    else:
        raise ValueError(f"Tipo de mensaje no reconocido: {tlv_type}")


def extract_bet(bet):
    index = 0
    elems = []
    while index < len(bet):
        length = int(bet[index + 1])
        index += TLV_HEADER_SIZE
        content = bet[index:index + length].decode("utf-8")
        elems.append(content)
        index += length

    return Bet(
        agency_id=0,
        first_name=elems[0],
        last_name=elems[1],
        document=int(elems[2]),
        birthdate=elems[3],
        number=int(elems[4]),
    )


def send_winners(socket, winners):
    print(f"[SERVER] enviando ganadores serializado: {winners}")
    safe_socket.send_all(socket, serialize_winners(winners))