package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/vpatsenko/pow-server/internal/pkg/config"
	"github.com/vpatsenko/pow-server/internal/pkg/pow"
	"github.com/vpatsenko/pow-server/internal/pkg/protocol"
)

func Run(cfg *config.Config) error {
	address := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	log.Println("connected to", address)
	defer conn.Close()

	for {
		message, err := HandleConnection(cfg, conn, conn)
		if err != nil {
			return err
		}
		log.Println("quote result:", message)
		time.Sleep(5 * time.Second)
	}
}

func HandleConnection(cfg *config.Config, readerConn io.Reader, writerConn io.Writer) (string, error) {
	reader := bufio.NewReader(readerConn)

	err := sendMsg(protocol.Message{
		Header: protocol.RequestChallenge,
	}, writerConn)
	if err != nil {
		return "", fmt.Errorf("err send request: %w", err)
	}

	msgStr, err := readConnMsg(reader)
	if err != nil {
		return "", fmt.Errorf("err read msg: %w", err)
	}
	msg, err := protocol.ParseMessage(msgStr)
	if err != nil {
		return "", fmt.Errorf("err parse msg: %w", err)
	}
	var hashcash pow.HashcashData
	err = json.Unmarshal([]byte(msg.Payload), &hashcash)
	if err != nil {
		return "", fmt.Errorf("err parse hashcash: %w", err)
	}
	log.Println("got hashcash:", hashcash)

	// conf := ctx.Value("cddonfig").(*config.Config)
	hashcash, err = hashcash.ComputeHashcash(cfg.HashcashMaxIterations)
	if err != nil {
		return "", fmt.Errorf("err compute hashcash: %w", err)
	}
	log.Println("hashcash computed:", hashcash)

	byteData, err := json.Marshal(hashcash)
	if err != nil {
		return "", fmt.Errorf("err marshal hashcash: %w", err)
	}

	err = sendMsg(protocol.Message{
		Header:  protocol.RequestResource,
		Payload: string(byteData),
	}, writerConn)
	if err != nil {
		return "", fmt.Errorf("err send request: %w", err)
	}
	log.Println("challenge sent to server")

	msgStr, err = readConnMsg(reader)
	if err != nil {
		return "", fmt.Errorf("err read msg: %w", err)
	}
	msg, err = protocol.ParseMessage(msgStr)
	if err != nil {
		return "", fmt.Errorf("err parse msg: %w", err)
	}
	return msg.Payload, nil
}

func readConnMsg(reader *bufio.Reader) (string, error) {
	return reader.ReadString('\n')
}

func sendMsg(msg protocol.Message, conn io.Writer) error {
	msgStr := fmt.Sprintf("%s\n", msg.Stringify())
	_, err := conn.Write([]byte(msgStr))
	return err
}
