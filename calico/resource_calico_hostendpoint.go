package calico

import (
	"fmt"
	"net"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/errors"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
)

func resourceCalicoHostendpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoHostendpointCreate,
		Read:   resourceCalicoHostendpointRead,
		Update: resourceCalicoHostendpointUpdate,
		Delete: resourceCalicoHostendpointDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
			"node": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"interface": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"expected_ips": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"profiles": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dToHostEndpointMetadata(d *schema.ResourceData) api.HostEndpointMetadata {
	metadata := api.HostEndpointMetadata{
		Name: d.Get("name").(string),
		Node: d.Get("node").(string),
	}

	if v, ok := d.GetOk("labels"); ok {
		labelMap := v.(map[string]interface{})
		labels := make(map[string]string, len(labelMap))

		for k, v := range labelMap {
			labels[k] = v.(string)
		}
		metadata.Labels = labels
	}

	return metadata
}

func dToHostEndpointSpec(d *schema.ResourceData) (api.HostEndpointSpec, error) {
	spec := api.HostEndpointSpec{}
	spec.InterfaceName = d.Get("interface").(string)

	if v, ok := d.GetOk("expected_ips.#"); ok {
		ips := make([]caliconet.IP, v.(int))

		for i := range ips {
			ip := d.Get("expected_ips." + strconv.Itoa(i)).(string)
			validIP := net.ParseIP(ip)
			if validIP == nil {
				return spec, fmt.Errorf("expected_ips: %v is not IP", ip)
			}
			ips[i] = caliconet.IP{validIP}
		}

		if len(ips) != 0 {
			spec.ExpectedIPs = ips
		}
	}

	if v, ok := d.GetOk("profiles.#"); ok {
		profiles := make([]string, v.(int))

		for i := range profiles {
			profiles[i] = d.Get("profiles." + strconv.Itoa(i)).(string)
		}

		if len(profiles) != 0 {
			spec.Profiles = profiles
		}
	}

	return spec, nil
}

func resourceCalicoHostendpointCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	metadata := dToHostEndpointMetadata(d)
	spec, err := dToHostEndpointSpec(d)
	if err != nil {
		return err
	}

	hostEndpoints := calicoClient.HostEndpoints()
	if _, err = hostEndpoints.Create(&api.HostEndpoint{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	d.SetId(metadata.Name)
	return resourceCalicoHostendpointRead(d, meta)
}

func resourceCalicoHostendpointRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	hostEndpoints := calicoClient.HostEndpoints()
	hostEndpoint, err := hostEndpoints.Get(api.HostEndpointMetadata{
		Name: d.Get("name").(string),
		Node: d.Get("node").(string),
	})

	// Handle endpoint does not exist
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	d.SetId(hostEndpoint.Metadata.Name)
	d.Set("name", hostEndpoint.Metadata.Name)
	d.Set("node", hostEndpoint.Metadata.Node)
	d.Set("labels", hostEndpoint.Metadata.Labels)

	d.Set("profiles", hostEndpoint.Spec.Profiles)

	ipList := make([]string, len(hostEndpoint.Spec.ExpectedIPs))
	for i, ip := range hostEndpoint.Spec.ExpectedIPs {
		ipList[i] = ip.String()
	}
	d.Set("expected_ips", ipList)
	d.Set("interface", hostEndpoint.Spec.InterfaceName)

	return nil
}

func resourceCalicoHostendpointUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	hostEndpoints := calicoClient.HostEndpoints()

	// Handle non-existant resource
	metadata := dToHostEndpointMetadata(d)
	if _, err := hostEndpoints.Get(metadata); err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	// Simply recreate the complete resource
	spec, err := dToHostEndpointSpec(d)
	if err != nil {
		return err
	}

	if _, err = hostEndpoints.Apply(&api.HostEndpoint{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	return nil
}

func resourceCalicoHostendpointDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	hostEndpoints := calicoClient.HostEndpoints()
	err := hostEndpoints.Delete(api.HostEndpointMetadata{
		Name: d.Get("name").(string),
		Node: d.Get("node").(string),
	})

	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	return nil
}
