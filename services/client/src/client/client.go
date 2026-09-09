// Package client implements the lottery client that reads bets from an
// input CSV, sends them to the lottery server in configurable batches and
// persists the winners to an output file.
package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/protocol"
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

	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		logger.Error("create-output-file", logger.Fail, "err", err)
		return err
	}
	defer outputFile.Close()

	reader := bufio.NewScanner(inputFile)
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	messageId := 0

	collectedBets := make([]protocol.Bet, 0, client.config.BatchSize)

	for reader.Scan() {
		messageId++
		messageArgs := []any{"agency-id", client.config.AgencyId, "message-id", messageId}
		logger.Info(mainAction, logger.InProgress, messageArgs...)

		clientMessage := reader.Text()

		bet, err := protocol.ParseBetFromCsv(clientMessage, client.config.AgencyId)
		if err != nil {
			return err
		}

		collectedBets = append(collectedBets, bet)

		logger.Info("send-message", logger.InProgress,
			"agency-id", client.config.AgencyId,
			"message-id", messageId,
		)

		if len(collectedBets) == client.config.BatchSize {
			if err := client.sendBatch(collectedBets, "send-message", messageArgs); err != nil {
				return err
			}
			collectedBets = collectedBets[:0]
		}

	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("error al leer los datos: %w", err)
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)

	if len(collectedBets) > 0 {
		messageArgs := []any{"agency-id", client.config.AgencyId, "remaining-count", len(collectedBets)}
		if err := client.sendBatch(collectedBets, "send-remaining-message", messageArgs); err != nil {
			return err
		}
		collectedBets = collectedBets[:0]
	}

	endBetsMessage, err := protocol.SerializeEndMessage(client.config.AgencyId)
	if err != nil {
		return err
	}

	if err := safe_socket.SendAll(client.conn, endBetsMessage); err != nil {
		return err
	}

	winners, err := protocol.DeserializeWinners(client.conn)
	if err != nil {
		return err
	}

	if err := storeWinners(writer, winners); err != nil {
		return err
	}

	return nil
}

// writes the `winners` slice to the corresponding output file.
// Each winner is written as a single line with the fields:
// FirstName,LastName,Id,Birthdate,BetNumber
// Returns an error if writing to `writer` fails.
func storeWinners(writer *bufio.Writer, winners []protocol.Bet) error {
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

// serializes the bets slice into a batch,
// sends it and waits for a server acknowledgement.
func (client *Client) sendBatch(bets []protocol.Bet, action string, messageArgs []any) error {
	if len(bets) == 0 {
		return nil
	}

	serializedBatch, err := protocol.SerializeBatch(bets)
	if err != nil {
		return err
	}

	if err := safe_socket.SendAll(client.conn, serializedBatch); err != nil {
		logger.Error(action, logger.Fail, messageArgs...)
		return err
	}

	if err := protocol.DeserializeAck(client.conn); err != nil {
		logger.Error("receive-ack", logger.Fail, "agency-id", client.config.AgencyId, "err", err)
		return err
	}

	logger.Info(action, logger.Success,
		"agency-id", client.config.AgencyId,
		"sent-bytes", len(serializedBatch),
	)
	return nil
}

// Close closes the network connection if it is open.
// It returns any error produced by `net.Conn.Close()`.
func (client *Client) Close() error {
	if client.conn != nil {
		return client.conn.Close()
	}
	return nil
}
