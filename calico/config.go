package calico

import (
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
)

type config struct {
	config api.CalicoAPIConfig
	Client *client.Client
}

func (c *config) loadAndValidate() error {
	calicoClient, err := client.New(c.config)
	c.Client = calicoClient

	return err
}
