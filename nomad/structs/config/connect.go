package config

import (
	"github.com/hashicorp/nomad/helper"
)

// ConnectSidecarConfig informs the Nomad Client how the Connect sidecar proxy
// image should be resolved.
//
// For now only envoy as a docker image is supported (see Consul). For non-envoy
// sidecar proxies, the sidecar_task of the service.connect must be set instead.
type ConnectSidecarConfig struct {
	// PullOnDemand dictates whether Nomad client will automatically pull the
	// latest envoy image that is known to be compatible with the version of
	// the local consul fingerprinted.
	PullOnDemand *bool `hcl:"pull_on_demand"`

	// Images is an optional allow-map of sidecar images that can be used.
	//
	// $ curl -s localhost:8500/v1/agent/self | jq .xDS
	//{
	//  "SupportedProxies": {
	//    "envoy": [
	//      "1.15.0",
	//      "1.14.4",
	//      "1.13.4",
	//      "1.12.6"
	//    ]
	//  }
	//}
	//
	// For now the only supported medium is envoy docker images, but this could
	// be expanded on in the future.
	Images *ConnectSidecarImages `hcl:"images"`
}

func (c *ConnectSidecarConfig) Merge(o *ConnectSidecarConfig) *ConnectSidecarConfig {
	result := o.Copy()

	if o.PullOnDemand != nil {
		result.PullOnDemand = helper.BoolToPtr(*o.PullOnDemand)
	}

	result.Images = c.Images.Merge(o.Images)

	return result
}

func (c *ConnectSidecarConfig) Copy() *ConnectSidecarConfig {
	if c == nil {
		return nil
	}

	return &ConnectSidecarConfig{
		PullOnDemand: helper.BoolToPtr(*c.PullOnDemand),
		Images:       c.Images.Copy(),
	}
}

// ConnectSidecarImages
type ConnectSidecarImages struct {
	// Envoy is supported
	Envoy map[string]string `hcl:"envoy"`
}

func (c *ConnectSidecarImages) Merge(o *ConnectSidecarImages) *ConnectSidecarImages {
	result := c.Copy()

	for k, v := range o.Envoy {
		result.Envoy[k] = v
	}

	return result
}

func (c *ConnectSidecarImages) Copy() *ConnectSidecarImages {
	if c == nil {
		return nil
	}

	return &ConnectSidecarImages{
		Envoy: helper.CopyMapStringString(c.Envoy),
	}
}
