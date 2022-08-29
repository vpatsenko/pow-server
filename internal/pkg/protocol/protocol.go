package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Quit = iota
	RequestChallenge
	ResponseChallenge
	RequestResource
	ResponseResource
)

type Message struct {
	Header  int
	Payload string
}

func (m *Message) Stringify() string {
	return fmt.Sprintf("%d|%s", m.Header, m.Payload)
}

func ParseMessage(str string) (*Message, error) {
	str = strings.TrimSpace(str)
	var msgType int
	parts := strings.Split(str, "|")
	if len(parts) < 1 || len(parts) > 2 {
		return nil, fmt.Errorf("message doesn't match protocol")
	}
	msgType, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("cannot parse header")
	}
	msg := Message{
		Header: msgType,
	}
	if len(parts) == 2 {
		msg.Payload = parts[1]
	}
	return &msg, nil
}
