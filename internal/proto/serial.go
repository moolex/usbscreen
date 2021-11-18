package proto

import (
	"strings"

	"github.com/pkg/errors"
	"go.bug.st/serial"
)

type Options = serial.Mode

func NewSerial(name string) *Serial {
	return &Serial{name: name}
}

type Serial struct {
	name string
	port serial.Port
}

func (s *Serial) Ports() ([]string, error) {
	return serial.GetPortsList()
}

func (s *Serial) Open(opts *Options) (serial.Port, error) {
	ports, err := s.Ports()
	if err != nil {
		return nil, err
	}

	var matched string
	for _, name := range ports {
		if strings.Contains(name, s.name) {
			matched = name
			break
		}
	}
	if matched == "" {
		return nil, errors.New("USB port not found")
	}

	port, err := serial.Open(matched, opts)
	if err != nil {
		return nil, err
	}

	s.port = port
	return s.port, nil
}

func (s *Serial) Close() error {
	return s.port.Close()
}

func (s *Serial) Read(p []byte) (n int, err error) {
	return s.port.Read(p)
}

func (s *Serial) Write(p []byte) (n int, err error) {
	return s.port.Write(p)
}
