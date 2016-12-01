package calico

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
)

// Provider is the provider for terraform
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"backend_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "etcdv2",
				Description: "Either etcdv2 or kubernetes",
			},
			"backend_etcd_scheme": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CALICO_BACKEND_ETCD_SCHEME", "http"),
				Description: "default: http",
			},
			"backend_etcd_authority": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CALICO_BACKEND_ETCD_AUTHORITY", "127.0.0.1:2379"),
				Description: "default: 127.0.0.1:2379",
			},
			"backend_etcd_endpoints": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "multiple etcd endpoints separated by comma",
			},
			"backend_etcd_username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Etcd username",
			},
			"backend_etcd_password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Etcd password",
			},
			"backend_etcd_keyfile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "File location keyfile",
			},
			"backend_etcd_certfile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "File location certfile",
			},
			"backend_etcd_cacertfile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "File location cacert",
			},
			"backend_k8s_configfile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K8sKubeconfigFile`",
			},
			"backend_k8s_server": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K8sServer",
			},
			"backend_k8s_clientcert": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K8sClientCertificate",
			},
			"backend_k8s_clientkey": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K8sClientKey",
			},
			"backend_k8s_ca": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K8sCertificateAuthority",
			},
			"backend_k8s_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K8sToken",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"calico_hostendpoint": resourceCalicoHostendpoint(),
			"calico_profile":      resourceCalicoProfile(),
			"calico_policy":       resourceCalicoPolicy(),
			"calico_ippool":       resourceCalicoIpPool(),
			"calico_bgppeer":      resourceCalicoBgpPeer(),
			"calico_node":         resourceCalicoNode(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	calicoConfig := api.CalicoAPIConfig{}

	backendType := d.Get("backend_type").(string)

	switch backendType {
	case "etcdv2":
		calicoConfig.Spec.DatastoreType = api.DatastoreType(backendType)

		calicoConfig.Spec.EtcdScheme = d.Get("backend_etcd_scheme").(string)
		calicoConfig.Spec.EtcdAuthority = d.Get("backend_etcd_authority").(string)
		calicoConfig.Spec.EtcdEndpoints = d.Get("backend_etcd_endpoints").(string)
		calicoConfig.Spec.EtcdUsername = d.Get("backend_etcd_username").(string)
		calicoConfig.Spec.EtcdPassword = d.Get("backend_etcd_password").(string)
		calicoConfig.Spec.EtcdKeyFile = d.Get("backend_etcd_keyfile").(string)
		calicoConfig.Spec.EtcdCertFile = d.Get("backend_etcd_certfile").(string)
		calicoConfig.Spec.EtcdCACertFile = d.Get("backend_etcd_cacertfile").(string)
	case "kubernetes":
		calicoConfig.Spec.DatastoreType = api.DatastoreType(backendType)

		calicoConfig.Spec.K8sKubeconfigFile = d.Get("backend_k8s_configfile").(string)
		calicoConfig.Spec.K8sServer = d.Get("backend_k8s_server").(string)
		calicoConfig.Spec.K8sClientCertificate = d.Get("backend_k8s_clientcert").(string)
		calicoConfig.Spec.K8sClientKey = d.Get("backend_k8s_clientkey").(string)
		calicoConfig.Spec.K8sCertificateAuthority = d.Get("backend_k8s_ca").(string)
		calicoConfig.Spec.K8sToken = d.Get("backend_k8s_token").(string)
	default:
		return nil, fmt.Errorf("backend_type etcdv2 is the only supported backend at the moment")
	}

	calicoClient, err := client.New(calicoConfig)
	if err != nil {
		return nil, err
	}

	config := config{
		config: calicoConfig,
		Client: calicoClient,
	}

	log.Printf("Configured: %#v", config)

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return config, nil
}
