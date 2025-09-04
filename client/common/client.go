package common

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
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
	msg := serializeBatch(batch, c.config.ID)
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

	for {
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
		c.conn.Close()
		time.Sleep(1 * time.Second)
		if err != nil {
			return 0, err
		}
		resp = strings.TrimSpace(resp)
		if resp == "ERROR" {
			continue
		}
		if resp == "" {
			return 0, nil
		}
		dnis := strings.Split(resp, "|")
		return len(dnis), nil
	}
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
	var bets [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) != 5 {
			continue
		}
		bets = append(bets, record)
	}
	return bets, nil
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

func serializeBatch(bets [][]string, agencyID string) string {
	lines := make([]string, 0, len(bets))
	for _, bet := range bets {
		line := append([]string{agencyID}, bet...)
		lines = append(lines, strings.Join(line, "|"))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (c *Client) StartClientLoop() {
	const maxBytes = 8192
	if c.file == nil {
		log.Criticalf("action: read_csv | result: fail | error: CSV file not opened")
		return
	}
	reader := csv.NewReader(c.file)
	reader.FieldsPerRecord = 5

	loopAmount := c.config.LoopAmount
	batchMax := c.config.BatchMax

	batch := make([][]string, 0, batchMax)
	batchSize := 0
	batchCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("action: read_csv | result: fail | error: %v", err)
			return
		}
		if len(record) != 5 {
			continue
		}
		betStr := strings.Join(record, "|")
		if batchSize+len(betStr)+1 > maxBytes || len(batch) >= batchMax {
			if len(batch) > 0 {
				if batchCount < loopAmount {
					if err := c.SendBatch(batch); err != nil {
						log.Errorf("action: send_batch | result: fail | error: %v", err)
						return
					}
					batchCount++
					time.Sleep(c.config.LoopPeriod)
				}
				batch = make([][]string, 0, batchMax)
				batchSize = 0
			}
		}
		batch = append(batch, record)
		batchSize += len(betStr) + 1
	}

	if len(batch) > 0 && batchCount < loopAmount {
		if err := c.SendBatch(batch); err != nil {
			log.Errorf("action: send_batch | result: fail | error: %v", err)
			return
		}
		batchCount++
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
