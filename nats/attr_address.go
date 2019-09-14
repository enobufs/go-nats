package nats

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/pion/stun"
)

const (
	familyIPv4 uint16 = 0x01
	familyIPv6 uint16 = 0x02
)

const (
	attrTypeChangeRequest  stun.AttrType = 0x0003 // CHANGE-REQUEST
	attrTypeChangedAddress stun.AttrType = 0x0005 // CHANGED-ADDRESS
	attrTypeOtherAddress   stun.AttrType = 0x802C // OTHER-ADDRESS
)

// attrAddress represents MAPPED-ADDRESS attribute.
//
// This attribute is used only by servers for achieving backwards
// compatibility with RFC 3489 clients.
//
// RFC 5389 Section 15.1
type attrAddress struct {
	IP   net.IP
	Port int
}

func (a attrAddress) String() string {
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
}

func (a *attrAddress) getAs(m *stun.Message, t stun.AttrType) error {
	v, err := m.Get(t)
	if err != nil {
		return err
	}
	if len(v) <= 4 {
		return io.ErrUnexpectedEOF
	}
	family := binary.BigEndian.Uint16(v[0:2])
	if family != familyIPv6 && family != familyIPv4 {
		return fmt.Errorf("xor-mapped address: bad family value %d", family)
	}
	ipLen := net.IPv4len
	if family == familyIPv6 {
		ipLen = net.IPv6len
	}
	// Ensuring len(a.IP) == ipLen and reusing a.IP.
	if len(a.IP) < ipLen {
		a.IP = a.IP[:cap(a.IP)]
		for len(a.IP) < ipLen {
			a.IP = append(a.IP, 0)
		}
	}
	a.IP = a.IP[:ipLen]
	for i := range a.IP {
		a.IP[i] = 0
	}
	a.Port = int(binary.BigEndian.Uint16(v[2:4]))
	copy(a.IP, v[4:])
	return nil
}

func (a *attrAddress) addAs(m *stun.Message, t stun.AttrType) error {
	var (
		family = familyIPv4
		ip     = a.IP
	)
	if len(a.IP) == net.IPv6len {
		if ip.To4() != nil {
			ip = ip[12:16] // like in ip.To4()
		} else {
			family = familyIPv6
		}
	} else if len(ip) != net.IPv4len {
		return fmt.Errorf("attrAddr: bad IPv4 length %d", len(ip))
	}
	value := make([]byte, 128)
	value[0] = 0 // first 8 bits are zeroes
	binary.BigEndian.PutUint16(value[0:2], family)
	binary.BigEndian.PutUint16(value[2:4], uint16(a.Port))
	copy(value[4:], ip)
	m.Add(t, value[:4+len(ip)])
	return nil
}
