package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  int
}

type Client struct {
	conn   net.Conn
	config ClientConfig
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	client := &Client{conn: conn, config: config}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	const mainAction = "test-echo-server"
	defer client.conn.Close()

	input_file, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}
	defer input_file.Close()

	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		logger.Error("create-output-file", logger.Fail, "err", err)
		return err
	}
	defer outputFile.Close()

	reader := bufio.NewScanner(input_file)
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	messageId := 0

	collected_bets := make([]Bet, 0, client.config.BatchSize)

	for reader.Scan() {
		messageId++
		messageArgs := []any{"agency-id", client.config.AgencyId, "message-id", messageId}
		logger.Info(mainAction, logger.InProgress, messageArgs...)

		clientMessage := reader.Text()

		bet, err := parseBetFromCsv(clientMessage, client.config.AgencyId)
		if err != nil {
			return err
		}

		collected_bets = append(collected_bets, bet)

		logger.Info("send-message", logger.InProgress,
			"agency-id", client.config.AgencyId,
			"message-id", messageId,
		)

		if len(collected_bets) == client.config.BatchSize {
			serializedBatch, err := serialize_batch(collected_bets)
			if err != nil {
				return err
			}

			if err := safe_socket.SendAll(client.conn, serializedBatch); err != nil {
				logger.Error("send-message", logger.Fail, messageArgs...)
				return err
			}

			if err := deserialize_ack(client.conn); err != nil {
				logger.Error("receive-ack", logger.Fail, "agency-id", client.config.AgencyId, "err", err)
				return err
			}

			collected_bets = collected_bets[:0]

			logger.Info("send-message", logger.Success,
				"agency-id", client.config.AgencyId,
				"message-id", messageId,
				"sent-bytes", len(serializedBatch),
			)
		}

	}
	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)

	if len(collected_bets) > 0 {
		serializedBatch, err := serialize_batch(collected_bets)
		if err != nil {
			return err
		}

		if err := safe_socket.SendAll(client.conn, serializedBatch); err != nil {
			logger.Error("send-remaining-message", logger.Fail, "agency-id", client.config.AgencyId, "err", err)
			return err
		}

		if err := deserialize_ack(client.conn); err != nil {
			logger.Error("receive-ack", logger.Fail, "agency-id", client.config.AgencyId, "err", err)
			return err
		}

		logger.Info("send-remaining-message", logger.Success,
			"agency-id", client.config.AgencyId,
			"sent-bytes", len(serializedBatch),
			"remaining-count", len(collected_bets),
		)

		collected_bets = collected_bets[:0]
	}

	endBetsMessage, err := serialize_end_message(client.config.AgencyId)
	if err != nil {
		return err
	}

	if err := safe_socket.SendAll(client.conn, endBetsMessage); err != nil {
		return err
	}

	winners, err := deserialize_winners(client.conn)
	if err != nil {
		return err
	}

	if err := storeWinners(writer, winners); err != nil {
		return err
	}

	return nil
}

func storeWinners(writer *bufio.Writer, winners []Bet) error {
	for _, winner := range winners {
		line := fmt.Sprintf("%s,%s,%s,%s,%s\n",
			winner.FirstName,
			winner.LastName,
			winner.Id,
			winner.Birthdate,
			winner.BetNumber,
		)
		_, err := writer.WriteString(line)
		if err != nil {
			logger.Error("write-output-file", logger.Fail, "err", err)
			return err
		}
	}

	return nil
}

func (client *Client) Close() error {
	if client.conn != nil {
		return client.conn.Close()
	}
	return nil
}
