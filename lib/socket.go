// 2019 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license
package lib

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/goshuirc/irc-go/ircmsg"
)

var (
	// ErrorDisconnected indicates that this socket is disconnected.
	ErrorDisconnected = errors.New("Socket is disconnected")
)

// Socket appropriately buffers IRC lines.
type Socket struct {
	connection net.Conn

	connectedMutex sync.Mutex
	connected      bool

	readMutex sync.Mutex
	reader    *bufio.Reader

	writeMutex sync.Mutex
	writer     *bufio.Writer
}

// ConnectSocket connects to the given host/port and starts our receivers if appropriate.
func ConnectSocket(address string, useTLS bool, tlsConfig *tls.Config) (*Socket, error) {
	var conn net.Conn
	var err error

	if useTLS {
		conn, err = tls.Dial("tcp", address, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		return nil, err
	}

	// set socket details
	s := Socket{
		connected:  true,
		connection: conn,
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
	}

	return &s, nil
}

// MakeSocket makes a socket from the given connection.
func MakeSocket(conn net.Conn) *Socket {
	return &Socket{
		connected:  true,
		connection: conn,
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
	}
}

// GetLine returns a single IRC line from the socket.
func (s *Socket) GetLine() (string, error) {
	if !s.Connected() {
		return "", ErrorDisconnected
	}

	s.readMutex.Lock()
	defer s.readMutex.Unlock()
	lineBytes, err := s.reader.ReadBytes('\n')

	return strings.TrimRight(string(lineBytes), "\r\n"), err
}

// SendLine sends a single IRC line to the socket
func (s *Socket) SendLine(line string) error {
	if !s.Connected() {
		return ErrorDisconnected
	}

	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	_, err := s.writer.WriteString(strings.TrimRight(line, "\r\n") + "\r\n")
	if err == nil {
		err = s.writer.Flush()
	}
	return err
}

// Send the given message.
func (s *Socket) Send(tags map[string]string, prefix string, command string, params ...string) error {
	msg := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := msg.Line()
	if err == nil {
		err = s.SendLine(line)
	}
	return err
}

// Disconnect severs our connection to the server.
func (s *Socket) Disconnect() {
	s.connectedMutex.Lock()
	defer s.connectedMutex.Unlock()

	if !s.connected {
		s.connected = false
		s.connection.Close()
	}
}

// Connected returns true if we're still connected
func (s *Socket) Connected() bool {
	s.connectedMutex.Lock()
	defer s.connectedMutex.Unlock()

	return s.connected
}
