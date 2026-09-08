import socket

# TODO: Complete with a short-read/short-write tolerant implementation

def recv_all(socket: socket.socket, size):
    bytes_received = 0
    buffer = bytearray()
    while bytes_received < size: 
        data = socket.recv(size - bytes_received)
        if not data:
            raise ConnectionError
        bytes_received += len(data)
        buffer.extend(data)
    return bytes(buffer)


def send_all(socket: socket.socket, bytes):
    bytes_sent = 0
    while bytes_sent < len(bytes): 
        bytes_written = socket.send(bytes[bytes_sent:])
        if bytes_written == 0:
            continue
        bytes_sent += bytes_written
    

    
