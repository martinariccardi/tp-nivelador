package safe_socket

import "io"

//TODO: Complete with a short-read/short-write tolerant implementation

func SendAll(socket io.Writer, bytes []byte) error {
	bytes_sent := 0
	for bytes_sent < len(bytes) {
		bytes_written, err := socket.Write(bytes[bytes_sent:])
		bytes_sent += bytes_written
		if err != nil {
			return err
		}
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	bytes_received := 0
	buff := make([]byte, size)
	for bytes_received < size {
		bytes_read, err := socket.Read(buff[bytes_received:])
		bytes_received += bytes_read
		if err != nil{
			if bytes_received == size{
				break
			}
			return nil, err
		}
	}
	return buff[:n], nil
}
