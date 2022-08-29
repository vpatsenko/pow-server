package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"

	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vpatsenko/pow-server/internal/pkg/cache"
	"github.com/vpatsenko/pow-server/internal/pkg/config"
	"github.com/vpatsenko/pow-server/internal/pkg/pow"
	"github.com/vpatsenko/pow-server/internal/pkg/protocol"
)

var Quotes = []string{
	`All saints who remember to keep and do these sayings,
	walking in obedience to the commandments,
	shall receive health in their navel and marrow to their bones`,

	`And shall find wisdom and great treasures of knowledge, even hidden treasures`,

	`And shall run and not be weary, and shall walk and not faint`,

	`And I, the Lord, give unto them a promise,
	that the destroying angel shall pass by them,
	as the children of Israel, and not slay them`,
}

var ErrQuit = errors.New("client requests to close connection")

type Server interface {
	Run(ctx context.Context) error
}

type server struct {
	cfg   *config.Config
	cache *cache.Cache
}

func NewServer(cfg *config.Config, cache *cache.Cache) Server {
	return &server{
		cfg:   cfg,
		cache: cache,
	}
}

func (s *server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.cfg.ServerHost, s.cfg.ServerPort))
	if err != nil {
		return err
	}

	defer listener.Close()
	log.Println("listening", listener.Addr())

	go func() {
		<-ctx.Done()
		log.Println("Shutting down the server")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("error accept connection: %w", err)
		}
		go s.handleConnection(ctx, conn)
	}
}

func (s *server) handleConnection(ctx context.Context, conn net.Conn) {
	log.Println("new client:", conn.RemoteAddr())
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		req, err := reader.ReadString('\n')
		if err != nil {
			log.Println("err read connection:", err)
			return
		}
		msg, err := s.processRequest(ctx, req, conn.RemoteAddr().String())
		if err != nil {
			log.Println("err process request:", err)
			return
		}
		if msg != nil {
			err := sendMsg(*msg, conn)
			if err != nil {
				log.Println("err send message:", err)
			}
		}
	}
}

func (s *server) processRequest(ctx context.Context, msgStr string, clientInfo string) (*protocol.Message, error) {
	msg, err := protocol.ParseMessage(msgStr)
	if err != nil {
		return nil, err
	}

	switch msg.Header {
	case protocol.Quit:
		return nil, ErrQuit
	case protocol.RequestChallenge:
		log.Printf("client %s requests challenge\n", clientInfo)

		randValue := rand.Intn(100000)
		err := s.cache.Add(randValue, s.cfg.HashcashDuration)
		if err != nil {
			return nil, errors.Wrap(err, "err add rand to cache:")
		}

		hashcash := pow.HashcashData{
			Version:    1,
			ZerosCount: s.cfg.HashcashZerosCount,
			Date:       time.Now().Unix(),
			Resource:   clientInfo,
			Rand:       base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", randValue))),
			Counter:    0,
		}
		hashcashMarshaled, err := json.Marshal(hashcash)
		if err != nil {
			return nil, fmt.Errorf("err marshal hashcash: %v", err)
		}
		msg := protocol.Message{
			Header:  protocol.ResponseChallenge,
			Payload: string(hashcashMarshaled),
		}
		return &msg, nil
	case protocol.RequestResource:
		log.Printf("client %s requests resource with payload %s\n", clientInfo, msg.Payload)
		var hashcash pow.HashcashData
		err := json.Unmarshal([]byte(msg.Payload), &hashcash)
		if err != nil {
			return nil, fmt.Errorf("err unmarshal hashcash: %w", err)
		}

		if hashcash.Resource != clientInfo {
			return nil, fmt.Errorf("invalid hashcash resource")
		}

		randValueBytes, err := base64.StdEncoding.DecodeString(hashcash.Rand)
		if err != nil {
			return nil, fmt.Errorf("err decode rand: %w", err)
		}
		randValue, err := strconv.Atoi(string(randValueBytes))
		if err != nil {
			return nil, fmt.Errorf("err decode rand: %w", err)
		}

		exists := s.cache.Get(randValue)
		if !exists {
			return nil, fmt.Errorf("challenge expired or not sent")
		}

		if time.Now().Unix()-hashcash.Date > s.cfg.HashcashDuration {
			return nil, fmt.Errorf("challenge expired")
		}

		maxIter := hashcash.Counter
		if maxIter == 0 {
			maxIter = 1
		}
		_, err = hashcash.ComputeHashcash(maxIter)
		if err != nil {
			return nil, fmt.Errorf("invalid hashcash")
		}

		log.Printf("client %s succesfully computed hashcash %s\n", clientInfo, msg.Payload)
		msg := protocol.Message{
			Header:  protocol.ResponseResource,
			Payload: Quotes[rand.Intn(4)],
		}

		s.cache.Delete(randValue)
		return &msg, nil
	default:
		return nil, fmt.Errorf("unknown header")
	}
}

func sendMsg(msg protocol.Message, conn net.Conn) error {
	msgStr := fmt.Sprintf("%s\n", msg.Stringify())
	_, err := conn.Write([]byte(msgStr))
	return err
}
