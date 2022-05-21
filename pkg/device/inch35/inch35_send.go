package inch35

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (i *Inch35) sendCMD(code uint8, vars ...int) error {
	if len(vars) > 4 {
		return errors.New("too many vars")
	}

	var vars2 [4]int
	for i, v := range vars {
		vars2[i] = v
	}

	return i.sendRaw(code, vars2[0], vars2[1], vars2[2], vars2[3], nil)
}

func (i *Inch35) sendOpt(code uint8, fixed int, bytes []byte) error {
	if len(bytes) > fixed {
		return errors.New("too many bytes")
	}

	bytes = append(make([]byte, 6), bytes...)
	if len(bytes) < fixed {
		bytes = append(bytes, make([]byte, fixed-len(bytes))...)
	}

	return i.sendRaw(code, 0, 0, 0, 0, bytes)
}

func (i *Inch35) sendRaw(code uint8, var1 int, var2 int, var3 int, var4 int, bytes []byte) error {
	if len(bytes) == 0 {
		bytes = make([]byte, 6)
	}

	bytes[0] = (byte)(var1 >> 2)
	bytes[1] = (byte)(((var1 & 3) << 6) + (var2 >> 4))
	bytes[2] = (byte)(((var2 & 0xF) << 4) + (var3 >> 6))
	bytes[3] = (byte)(((var3 & 0x3F) << 2) + (var4 >> 8))
	bytes[4] = (byte)(var4 & 0xFF)
	bytes[5] = code

	return i.sendBytes(bytes)
}

func (i *Inch35) sendBytes(bytes []byte) error {
	var sent int
	var cost time.Duration

	start := time.Now()
	if n, err := i.serial.Write(bytes); err != nil {
		return err
	} else {
		sent = n
		cost = time.Since(start)
	}

	ext := ""
	if len(bytes) <= 16 {
		ext = fmt.Sprintf("%x", bytes)
	}

	i.logger.With(
		zap.Int("sent", sent),
		zap.String("cost", cost.String()),
		zap.String("data", ext),
	).Debug("transfer")

	return nil
}
