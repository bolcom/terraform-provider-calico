package calico

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/errors"
)

func resourceCalicoIpPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoIpPoolCreate,
		Read:   resourceCalicoIpPoolRead,
		Update: resourceCalicoIpPoolUpdate,
		Delete: resourceCalicoIpPoolDelete,

		Schema: map[string]*schema.Schema{
			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"spec": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipip": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
						"nat-outgoing": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"disabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func dToIpPoolMetadata(d *schema.ResourceData) (api.IPPoolMetadata, error) {
	metadata := api.IPPoolMetadata{}

	cidr, err := dToCIDR(d, "cidr")
	if err != nil {
		return metadata, err
	}
	metadata.CIDR = cidr

	return metadata, nil
}

func dToIpPoolSpec(d *schema.ResourceData) (api.IPPoolSpec, error) {
	spec := api.IPPoolSpec{}

	ipipEnabled := d.Get("spec.0.ipip.0.enabled").(bool)
	ipip := api.IPIPConfiguration{
		Enabled: ipipEnabled,
	}
	spec.IPIP = &ipip

	natOutgoing := d.Get("spec.0.nat-outgoing").(bool)
	spec.NATOutgoing = natOutgoing

	disabled := d.Get("spec.0.disabled").(bool)
	spec.Disabled = disabled

	return spec, nil
}

// set Schema Fields based on existing IPPool Specs
func setSchemaFieldsForIPPoolSpec(ippool *api.IPPool, d *schema.ResourceData) {
	specArray := make([]interface{}, 1)

	specMap := make(map[string]interface{})

	specMap["nat-outgoing"] = ippool.Spec.NATOutgoing
	specMap["disabled"] = ippool.Spec.Disabled

	ipipMapArray := make([]interface{}, 1)

	ipipMap := make(map[string]interface{})

	pIPIP := ippool.Spec.IPIP
	if pIPIP != nil {
		ipipMap["enabled"] = pIPIP.Enabled
		ipipMapArray[0] = ipipMap

		specMap["ipip"] = ipipMapArray
	}

	specArray[0] = specMap

	d.Set("spec", specArray)
}

func resourceCalicoIpPoolCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	metadata, err := dToIpPoolMetadata(d)
	if err != nil {
		return err
	}
	spec, err := dToIpPoolSpec(d)
	if err != nil {
		return err
	}

	ipPools := calicoClient.IPPools()
	if _, err = ipPools.Create(&api.IPPool{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	d.SetId(metadata.CIDR.String())
	return resourceCalicoIpPoolRead(d, meta)
}

func resourceCalicoIpPoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	ipPools := calicoClient.IPPools()
	cidr, err := dToCIDR(d, "cidr")
	if err != nil {
		return err
	}
	ipPool, err := ipPools.Get(api.IPPoolMetadata{
		CIDR: cidr,
	})

	// Handle endpoint does not exist
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	d.SetId(ipPool.Metadata.CIDR.String())
	d.Set("cidr", ipPool.Metadata.CIDR.String())
	setSchemaFieldsForIPPoolSpec(ipPool, d)

	return nil
}

func resourceCalicoIpPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	ipPools := calicoClient.IPPools()

	// Handle non-existant resource
	metadata, err := dToIpPoolMetadata(d)
	if err != nil {
		return err
	}
	if _, err := ipPools.Get(metadata); err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	// Simply recreate the complete resource
	spec, err := dToIpPoolSpec(d)
	if err != nil {
		return err
	}

	if _, err = ipPools.Apply(&api.IPPool{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	return nil
}

func resourceCalicoIpPoolDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	ipPools := calicoClient.IPPools()
	cidr, err := dToCIDR(d, "cidr")
	if err != nil {
		return err
	}
	err = ipPools.Delete(api.IPPoolMetadata{
		CIDR: cidr,
	})

	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	return nil
}
