package nats

import (
	"encoding/json"
	"testing"

	"github.com/pion/logging"
	"github.com/pion/transport/vnet"
	"github.com/stretchr/testify/assert"
)

type virtualNet struct {
	wan    *vnet.Router
	net0   *vnet.Net
	net1   *vnet.Net
	server *STUNServer
}

func (v *virtualNet) close() {
	v.server.Close() // nolint:errcheck,gosec
	v.wan.Stop()     // nolint:errcheck,gosec
}

func buildVNet(natType *vnet.NATType) (*virtualNet, error) {
	loggerFactory := logging.NewDefaultLoggerFactory()

	// WAN
	wan, err := vnet.NewRouter(&vnet.RouterConfig{
		CIDR:          "0.0.0.0/0",
		LoggerFactory: loggerFactory,
	})
	if err != nil {
		return nil, err
	}

	wanNet := vnet.NewNet(&vnet.NetConfig{
		StaticIPs: []string{"1.2.3.4", "1.2.3.5"},
	})

	err = wan.AddNet(wanNet)
	if err != nil {
		return nil, err
	}

	err = wan.AddHost("stun.pion.net", "1.2.3.4")
	if err != nil {
		return nil, err
	}

	// LAN 0
	lan0, err := vnet.NewRouter(&vnet.RouterConfig{
		StaticIP:      "27.1.1.1", // this router's external IP on eth0
		CIDR:          "192.168.0.0/24",
		NATType:       natType,
		LoggerFactory: loggerFactory,
	})
	if err != nil {
		return nil, err
	}

	net0 := vnet.NewNet(&vnet.NetConfig{})
	err = lan0.AddNet(net0)
	if err != nil {
		return nil, err
	}

	err = wan.AddRouter(lan0)
	if err != nil {
		return nil, err
	}

	// Start routers
	err = wan.Start()
	if err != nil {
		return nil, err
	}

	// Run STUN server
	server, err := NewSTUNServer(&STUNServerConfig{
		PrimaryAddress:   "1.2.3.4:3478",
		SecondaryAddress: "1.2.3.5:3479",
		Net:              wanNet,
		LoggerFactory:    loggerFactory,
	})
	if err != nil {
		return nil, err
	}

	err = server.Start()
	if err != nil {
		return nil, err
	}

	return &virtualNet{
		wan:    wan,
		net0:   net0,
		server: server,
	}, nil
}

func TestDiscoverOnVNet(t *testing.T) {
	loggerFactory := logging.NewDefaultLoggerFactory()
	log := loggerFactory.NewLogger("test")

	/*
		t.Run("stun.sipgate.net", func(t *testing.T) {
			res, err := Discover("stun.sipgate.net:3478", true)
			assert.NoError(t, err, "should succeed")
			assert.NotNil(t, res, "should not be nil")
		})
	*/

	t.Run("Full cone NAT", func(t *testing.T) {
		v, err := buildVNet(&vnet.NATType{
			MappingBehavior:   vnet.EndpointIndependent,
			FilteringBehavior: vnet.EndpointIndependent,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		defer v.close()

		nats, err := NewNATS(&Config{
			Server:  "stun.pion.net:3478",
			Verbose: true,
			Net:     v.net0,
		})
		assert.NoError(t, err, "should succeed")

		res, err := nats.Discover()
		assert.NoError(t, err, "should succeed")
		assert.NotNil(t, res, "should not be nil")

		bytes, err := json.MarshalIndent(res, "", "  ")
		assert.NoError(t, err, "should succeed")
		log.Debug(string(bytes))

		assert.True(t, res.IsNatted, "should be natted")
		assert.Equal(t, EndpointIndependent, res.MappingBehavior, "should match")
		assert.Equal(t, EndpointIndependent, res.FilteringBehavior, "should match")
		assert.False(t, res.PortPreservation, "should not be port preserved")
		assert.Equal(t, "Full cone NAT", res.NATType, "should match")
		assert.Equal(t, "27.1.1.1", res.ExternalIP, "should match")
	})

	t.Run("Restricted cone NAT", func(t *testing.T) {
		v, err := buildVNet(&vnet.NATType{
			MappingBehavior:   vnet.EndpointIndependent,
			FilteringBehavior: vnet.EndpointAddrDependent,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		defer v.close()

		nats, err := NewNATS(&Config{
			Server:  "stun.pion.net:3478",
			Verbose: true,
			Net:     v.net0,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}

		res, err := nats.Discover()
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		assert.NotNil(t, res, "should not be nil")

		bytes, err := json.MarshalIndent(res, "", "  ")
		assert.NoError(t, err, "should succeed")
		log.Debug(string(bytes))

		log.Debugf("%+v", res)
		assert.True(t, res.IsNatted, "should be natted")
		assert.Equal(t, EndpointIndependent, res.MappingBehavior, "should match")
		assert.Equal(t, EndpointAddrDependent, res.FilteringBehavior, "should match")
		assert.False(t, res.PortPreservation, "should not be port preserved")
		assert.Equal(t, "Address-restricted cone NAT", res.NATType, "should match")
		assert.Equal(t, "27.1.1.1", res.ExternalIP, "should match")
	})

	t.Run("Port-restricted cone NAT", func(t *testing.T) {
		v, err := buildVNet(&vnet.NATType{
			MappingBehavior:   vnet.EndpointIndependent,
			FilteringBehavior: vnet.EndpointAddrPortDependent,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		defer v.close()

		nats, err := NewNATS(&Config{
			Server:  "stun.pion.net:3478",
			Verbose: true,
			Net:     v.net0,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}

		res, err := nats.Discover()
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		assert.NotNil(t, res, "should not be nil")

		bytes, err := json.MarshalIndent(res, "", "  ")
		assert.NoError(t, err, "should succeed")
		log.Debug(string(bytes))

		log.Debugf("%+v", res)
		assert.True(t, res.IsNatted, "should be natted")
		assert.Equal(t, EndpointIndependent, res.MappingBehavior, "should match")
		assert.Equal(t, EndpointAddrPortDependent, res.FilteringBehavior, "should match")
		assert.False(t, res.PortPreservation, "should not be port preserved")
		assert.Equal(t, "Port-restricted cone NAT", res.NATType, "should match")
		assert.Equal(t, "27.1.1.1", res.ExternalIP, "should match")
	})

	t.Run("Symmetric NAT", func(t *testing.T) {
		v, err := buildVNet(&vnet.NATType{
			MappingBehavior:   vnet.EndpointAddrPortDependent,
			FilteringBehavior: vnet.EndpointAddrPortDependent,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		defer v.close()

		nats, err := NewNATS(&Config{
			Server:  "stun.pion.net:3478",
			Verbose: true,
			Net:     v.net0,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}

		res, err := nats.Discover()
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		assert.NotNil(t, res, "should not be nil")

		bytes, err := json.MarshalIndent(res, "", "  ")
		assert.NoError(t, err, "should succeed")
		log.Debug(string(bytes))

		log.Debugf("%+v", res)
		assert.True(t, res.IsNatted, "should be natted")
		assert.Equal(t, EndpointAddrPortDependent, res.MappingBehavior, "should match")
		assert.Equal(t, EndpointAddrPortDependent, res.FilteringBehavior, "should match")
		assert.False(t, res.PortPreservation, "should not be port preserved")
		assert.Equal(t, "Symmetric NAT", res.NATType, "should match")
		assert.Equal(t, "27.1.1.1", res.ExternalIP, "should match")
	})

	t.Run("Symmetric NAT 2", func(t *testing.T) {
		v, err := buildVNet(&vnet.NATType{
			MappingBehavior:   vnet.EndpointAddrDependent,
			FilteringBehavior: vnet.EndpointAddrPortDependent,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		defer v.close()

		nats, err := NewNATS(&Config{
			Server:  "stun.pion.net:3478",
			Verbose: true,
			Net:     v.net0,
		})
		if !assert.NoError(t, err, "should succeed") {
			return
		}

		res, err := nats.Discover()
		if !assert.NoError(t, err, "should succeed") {
			return
		}
		assert.NotNil(t, res, "should not be nil")

		bytes, err := json.MarshalIndent(res, "", "  ")
		assert.NoError(t, err, "should succeed")
		log.Debug(string(bytes))

		log.Debugf("%+v", res)
		assert.True(t, res.IsNatted, "should be natted")
		assert.Equal(t, EndpointAddrDependent, res.MappingBehavior, "should match")
		assert.Equal(t, EndpointAddrPortDependent, res.FilteringBehavior, "should match")
		assert.False(t, res.PortPreservation, "should not be port preserved")
		assert.Equal(t, "Symmetric NAT", res.NATType, "should match")
		assert.Equal(t, "27.1.1.1", res.ExternalIP, "should match")
	})
}
