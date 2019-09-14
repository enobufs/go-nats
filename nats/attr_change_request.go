package nats

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pion/stun"
)

// attrChangeRequest represents CHANGE-REQUEST attribute.
type attrChangeRequest struct {
	ChangeIP   bool
	ChangePort bool
}

func (a *attrChangeRequest) String() string {
	return fmt.Sprintf("changeIP=%v changePort=%v", a.ChangeIP, a.ChangePort)
}

func (a *attrChangeRequest) getAs(m *stun.Message, t stun.AttrType) error {
	bytes, err := m.Get(t)
	if err != nil {
		return err
	}
	if len(bytes) <= 4 {
		return io.ErrUnexpectedEOF
	}
	val := binary.BigEndian.Uint32(bytes[0:4])
	a.ChangeIP = val&0x4 != 0
	a.ChangePort = val&0x2 != 0
	return nil
}

func (a *attrChangeRequest) addAs(m *stun.Message, t stun.AttrType) error {
	var val uint32
	if a.ChangeIP {
		val |= 0x4
	}
	if a.ChangePort {
		val |= 0x2
	}
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, val)
	m.Add(t, bytes)
	return nil
}
