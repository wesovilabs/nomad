package taskrunner

import (
	"context"
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	ifs "github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/client/consul"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/pkg/errors"
)

const (
	// envoyVersionHookName is the name of this hook and appears in logs.
	envoyVersionHookName = "envoy_version"

	// envoyLegacyImage is used when the version of Consul is too old to support
	// the SupportedProxies field in the self API.
	//
	// This is the version defaulted by Nomad before v1.0.
	envoyLegacyImage = "envoyproxy/envoy:v1.11.2@sha256:a7769160c9c1a55bb8d07a3b71ce5d64f72b1f665f10d81aa1581bc3cf850d09"
)

type envoyVersionHookConfig struct {
	alloc         *structs.Allocation
	proxiesClient consul.SupportedProxiesAPI
	logger        hclog.Logger
}

func newEnvoyVersionHookConfig(alloc *structs.Allocation, proxiesClient consul.SupportedProxiesAPI, logger hclog.Logger) *envoyVersionHookConfig {
	return &envoyVersionHookConfig{
		alloc:         alloc,
		logger:        logger,
		proxiesClient: proxiesClient,
	}
}

type envoyVersionHook struct {
	// alloc is the allocation with the envoy task being rewritten.
	alloc *structs.Allocation

	// proxiesClient is the subset of the Consul API for getting information
	// from Consul about the versions of Envoy it supports.
	proxiesClient consul.SupportedProxiesAPI

	// logger is used to log things.
	logger hclog.Logger
}

func newEnvoyVersionHook(c *envoyVersionHookConfig) *envoyVersionHook {
	return &envoyVersionHook{
		alloc:         c.alloc,
		proxiesClient: c.proxiesClient,
		logger:        c.logger.Named(envoyVersionHookName),
	}
}

func (envoyVersionHook) Name() string {
	return envoyVersionHookName
}

// todo: determine what version of envoy to run
//  which is going to involve image, meta, consul version, consul api response

func (h *envoyVersionHook) Prestart(ctx context.Context, request *ifs.TaskPrestartRequest, response *ifs.TaskPrestartResponse) error {
	fmt.Println("EVH Prestart")

	if h.skip(request) {
		fmt.Println("skip is true")
		response.Done = true
		return nil
	}

	fmt.Println("skip is false")

	// it's either legcay or managable, need to know consul version
	proxies, err := h.proxiesClient.Proxies()
	if err != nil {
		return err
	}

	if proxies == nil {
		fmt.Println("proxies result is nil, use fallback")
	} else {
		fmt.Println("proxies:", proxies)
	}

	// todo obvious
	response.Done = true
	return errors.New("not yet finished")
}

// skip will return true if the request does not contain a task that should have
// its envoy proxy version resolved automatically.
func (h *envoyVersionHook) skip(request *ifs.TaskPrestartRequest) bool {
	switch {
	case request.Task.Driver != "docker": // we only manage docker
		return true
	case !request.Task.UsesConnectSidecar():
		return true
	case !h.isSentinel(request.Task.Config):
		return true
	}
	return false
}

func (_ *envoyVersionHook) isSentinel(config map[string]interface{}) bool {
	if len(config) == 0 {
		return false
	}

	image, ok := config["image"].(string)
	if !ok {
		return false
	}

	return image == structs.ConnectEnvoySentinel
}
