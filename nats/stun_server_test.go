package nats

import (
	"fmt"
	"net"
	"strings"

	"github.com/pion/logging"
	"github.com/pion/stun"
	"github.com/pion/transport/vnet"
)

type STUNServerConfig struct {
	PrimaryAddress   string
	SecondaryAddress string
	Net              *vnet.Net
	LoggerFactory    logging.LoggerFactory
}

type STUNServer struct {
	addrs    [4]*net.UDPAddr
	conns    [4]net.PacketConn
	software stun.Software
	net      *vnet.Net
	log      logging.LeveledLogger
}

func NewSTUNServer(config *STUNServerConfig) (*STUNServer, error) {
	log := config.LoggerFactory.NewLogger("stun-serv")

	pri := strings.Split(config.PrimaryAddress, ":")
	if len(pri) < 2 {
		pri = append(pri, "3478")
	}
	sec := strings.Split(config.SecondaryAddress, ":")
	if len(sec) < 2 {
		sec = append(sec, "3478")
	}

	if config.Net == nil {
		config.Net = vnet.NewNet(nil)
	}

	addrs := [4]*net.UDPAddr{}

	var err error
	addrs[0], err = config.Net.ResolveUDPAddr(
		"udp", fmt.Sprintf("%s:%s", pri[0], pri[1]))
	if err != nil {
		return nil, err
	}

	addrs[1], err = config.Net.ResolveUDPAddr(
		"udp", fmt.Sprintf("%s:%s", pri[0], sec[1]))
	if err != nil {
		return nil, err
	}

	addrs[2], err = config.Net.ResolveUDPAddr(
		"udp", fmt.Sprintf("%s:%s", sec[0], pri[1]))
	if err != nil {
		return nil, err
	}

	addrs[3], err = config.Net.ResolveUDPAddr(
		"udp", fmt.Sprintf("%s:%s", sec[0], sec[1]))
	if err != nil {
		return nil, err
	}

	return &STUNServer{addrs: addrs, net: config.Net, log: log}, nil
}

func (s *STUNServer) Start() error {
	for i, addr := range s.addrs {
		var err error
		s.log.Debugf("start listening on %s...", addr.String())
		s.conns[i], err = s.net.ListenUDP("udp", addr)
		if err != nil {
			return err
		}

		go s.readLoop(i)
	}
	return nil
}

func (s *STUNServer) readLoop(index int) {
	conn := s.conns[index]
	for {
		buf := make([]byte, 1500)
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			s.log.Errorf("readLoop: %s", err.Error())
			return
		}

		s.log.Debugf("received %d bytes from %s", n, from.String())

		m := &stun.Message{Raw: append([]byte{}, buf[:n]...)}
		if err = m.Decode(); err != nil {
			s.log.Warnf("failed to decode: %s", err.Error())
			continue
		}

		if m.Type.Class != stun.ClassRequest {
			s.log.Warn("not a request. dropping...")
			continue
		}

		if m.Type.Method != stun.MethodBinding {
			s.log.Warn("not a binding request. dropping...")
			continue
		}

		err = s.handleBindingRequest(index, from, m)
		if err != nil {
			s.log.Errorf("readLoop: handleBindingRequest failed: %s", err.Error())
			return
		}
	}
}

func (s *STUNServer) handleBindingRequest(index int, from net.Addr, m *stun.Message) error {
	s.log.Debugf("received BindingRequest from %s", from.String())

	var conn net.PacketConn

	// Check CHANGE-REQUEST
	changeReq := attrChangeRequest{}
	err := changeReq.GetFrom(m)
	if err != nil {
		s.log.Debugf("CHANGE-REQUEST not found: %s", err.Error())
		conn = s.conns[index]
	} else {
		s.log.Debugf("CHANGE-REQUEST: changeIP=%v changePort=%v",
			changeReq.ChangeIP, changeReq.ChangePort)
		if changeReq.ChangeIP {
			index ^= 0x2
		}
		if changeReq.ChangePort {
			index ^= 0x1
		}
		conn = s.conns[index]
	}

	udpAddr := from.(*net.UDPAddr)

	attrs := s.makeAttrs(m.TransactionID, stun.BindingSuccess,
		&stun.XORMappedAddress{
			IP:   udpAddr.IP,
			Port: udpAddr.Port,
		},
		&stun.XORMappedAddress{
			IP:   udpAddr.IP,
			Port: udpAddr.Port,
		},
		&attrChangedAddress{
			attrAddress{
				IP:   s.addrs[3].IP,
				Port: s.addrs[3].Port,
			},
		},
		stun.Fingerprint)

	msg, err := stun.Build(attrs...)
	if err != nil {
		return err
	}

	_, err = conn.WriteTo(msg.Raw, from)
	if err != nil {
		return err
	}
	return nil
}

func (s *STUNServer) makeAttrs(
	transactionID [stun.TransactionIDSize]byte,
	msgType stun.MessageType,
	additional ...stun.Setter) []stun.Setter {
	attrs := append([]stun.Setter{&stun.Message{TransactionID: transactionID}, msgType}, additional...)
	if len(s.software) > 0 {
		attrs = append(attrs, s.software)
	}
	return attrs
}

func (s *STUNServer) Close() error {
	var err error
	for _, conn := range s.conns {
		if conn != nil {
			err2 := conn.Close()
			if err2 != nil && err == nil {
				err = err2
			}
		}
	}
	return err
}
