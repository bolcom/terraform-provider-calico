package main

import (
	"github.com/bolcom/terraform-provider-calico/calico"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: calico.Provider,
	})
}
