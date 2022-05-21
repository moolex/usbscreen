package proto

import (
	"strings"

	"github.com/pkg/errors"
	"go.bug.st/serial"
)

type Options struct {
	DTR      bool
	RTS      bool
	BaudRate int
}

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

func (s *Serial) Open(opts *Options) error {
	ports, err := s.Ports()
	if err != nil {
		return err
	}

	var matched string
	for _, name := range ports {
		if strings.Contains(name, s.name) {
			matched = name
			break
		}
	}
	if matched == "" {
		return errors.New("USB port not found")
	}

	port, err := serial.Open(matched, &serial.Mode{BaudRate: opts.BaudRate})
	if err != nil {
		return err
	}

	if err := port.SetDTR(opts.DTR); err != nil {
		return err
	}

	if err := port.SetRTS(opts.RTS); err != nil {
		return err
	}

	s.port = port
	return nil
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
