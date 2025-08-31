package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
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

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.conn = conn
	return nil
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func serializeBet() (string, string, string) {
	nombre := getEnv("NOMBRE")
	apellido := getEnv("APELLIDO")
	documento := getEnv("DOCUMENTO")
	nacimiento := getEnv("NACIMIENTO")
	numero := getEnv("NUMERO")
	log.Infof("Enviando apuesta: %s|%s|%s|%s|%s", nombre, apellido, documento, nacimiento, numero)
	return fmt.Sprintf("%s|%s|%s|%s|%s\n", nombre, apellido, documento, nacimiento, numero), documento, numero
}

func (c *Client) StartClientLoop() {

	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {

		c.createClientSocket()

		msg, documento, numero := serializeBet()
		totalSent := 0
		for totalSent < len(msg) {
			n, err := c.conn.Write([]byte(msg)[totalSent:])
			if err != nil {
				log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
				c.conn.Close()
				return
			}
			totalSent += n
		}

		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.conn.Close()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s", documento, numero)

		time.Sleep(c.config.LoopPeriod)

	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
