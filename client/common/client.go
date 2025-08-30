package common

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	BatchMax      int
}

type Client struct {
	config ClientConfig
	conn   net.Conn
	file   *os.File
}

func (c *Client) SendBatch(batch [][]string) error {
	msg := serializeBatch(batch)
	if err := c.createClientSocket(); err != nil {
		return err
	}
	defer c.conn.Close()

	totalSent := 0
	msgBytes := []byte(msg)
	for totalSent < len(msgBytes) {
		n, err := c.conn.Write(msgBytes[totalSent:])
		if err != nil {
			return err
		}
		totalSent += n
	}

	resp, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return err
	}
	resp = strings.TrimSpace(resp)
	if resp != "OK" {
		return fmt.Errorf("server responded with %s", resp)
	}
	log.Infof("action: apuesta_enviada | result: success | cantidad: %d", len(batch))
	return nil
}

func (c *Client) NotifyFin() error {
	if err := c.createClientSocket(); err != nil {
		return err
	}
	defer c.conn.Close()
	finMsg := fmt.Sprintf("FIN|%s\n", c.config.ID)
	_, err := c.conn.Write([]byte(finMsg))
	if err != nil {
		return err
	}
	resp, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil || strings.TrimSpace(resp) != "OK" {
		return fmt.Errorf("server responded with %s", resp)
	}
	return nil
}

func (c *Client) ConsultarGanadores() (int, error) {
	if err := c.createClientSocket(); err != nil {
		return 0, err
	}
	defer c.conn.Close()
	consultaMsg := fmt.Sprintf("CONSULTA_GANADORES|%s\n", c.config.ID)
	_, err := c.conn.Write([]byte(consultaMsg))
	if err != nil {
		return 0, err
	}
	resp, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return 0, err
	}
	resp = strings.TrimSpace(resp)
	if resp == "ERROR" || resp == "" {
		return 0, nil
	}
	dnis := strings.Split(resp, "|")
	return len(dnis), nil
}

func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

func CloseClient(client *Client) {
	if client.conn != nil {
		client.conn.Close()
		log.Infof("action: shutdown | result: success | msg: Client socket closed")
	}
}

func (c *Client) OpenCSV(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	c.file = file
	return nil
}

func (c *Client) CloseCSV() {
	if c.file != nil {
		c.file.Close()
		log.Infof("action: shutdown | result: success | msg: CSV file closed")
	}
}

func (c *Client) ReadBetsFromCSV() ([][]string, error) {
	if c.file == nil {
		return nil, fmt.Errorf("CSV file not opened")
	}
	reader := csv.NewReader(c.file)
	reader.FieldsPerRecord = 5
	return reader.ReadAll()
}

func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return err
	}
	c.conn = conn
	return nil
}

func batchBets(bets [][]string, maxAmount int, maxBytes int) [][][]string {
	var batches [][][]string
	i := 0
	for i < len(bets) {
		var batch [][]string
		batchSize := 0
		for j := 0; j < maxAmount && i+j < len(bets); j++ {
			bet := bets[i+j]
			betStr := strings.Join(bet, "|")
			if batchSize+len(betStr)+1 > maxBytes {
				break
			}
			batch = append(batch, bet)
			batchSize += len(betStr) + 1
		}
		if len(batch) == 0 {

			batch = append(batch, bets[i])
			i++
		} else {
			i += len(batch)
		}
		batches = append(batches, batch)
	}
	return batches
}

func serializeBatch(bets [][]string) string {
	lines := make([]string, 0, len(bets))
	for _, bet := range bets {
		lines = append(lines, strings.Join(bet, "|"))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (c *Client) StartClientLoop() {
	bets, err := c.ReadBetsFromCSV()
	if err != nil {
		log.Criticalf("action: read_csv | result: fail | error: %v", err)
		return
	}

	const maxBytes = 8192
	batches := batchBets(bets, c.config.BatchMax, maxBytes)

	for _, batch := range batches {
		if err := c.SendBatch(batch); err != nil {
			log.Errorf("action: send_batch | result: fail | error: %v", err)
			return
		}
		time.Sleep(c.config.LoopPeriod)
	}

	if err := c.NotifyFin(); err != nil {
		log.Errorf("action: notify_fin | result: fail | error: %v", err)
		return
	}

	cantGanadores, err := c.ConsultarGanadores()
	if err != nil {
		log.Errorf("action: consulta_ganadores | result: fail | error: %v", err)
		return
	}
	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", cantGanadores)
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
