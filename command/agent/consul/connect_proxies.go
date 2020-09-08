package consul

import (
	"fmt"
)

// ConnectProxies implements SupportedProxiesAPI by using the Consul Agent API.
type ConnectProxies struct {
	agentAPI AgentAPI
}

func NewConnectProxiesClient(agentAPI AgentAPI) *ConnectProxies {
	return &ConnectProxies{
		agentAPI: agentAPI,
	}
}

// Proxies returns a map of the supported proxies.
func (c *ConnectProxies) Proxies() (map[string][]string, error) {
	self, err := c.agentAPI.Self()
	if err != nil {
		return nil, err
	}

	// todo decode
	_ = self

	fmt.Println("Proxies:", self)

	return map[string][]string{
		"envoy": []string{"v0", "v1"},
	}, nil
}
