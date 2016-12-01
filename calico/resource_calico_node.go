package calico

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/errors"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

func resourceCalicoNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoNodeCreate,
		Read:   resourceCalicoNodeRead,
		Update: resourceCalicoNodeUpdate,
		Delete: resourceCalicoNodeDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"spec": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bgp": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"asNumber": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"ipv4Address": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"ipv6Address": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dToNodeMetadata(d *schema.ResourceData) api.NodeMetadata {
	metadata := api.NodeMetadata{}

	metadata.Name = d.Get("name").(string)

	return metadata
}

func dToNodeSpec(d *schema.ResourceData) (api.NodeSpec, error) {
	spec := api.NodeSpec{}
	bgpSpec := api.NodeBGPSpec{}

	asNumber := d.Get("spec.0.bgp.0.asNumber").(string)

	num, err := numorstring.ASNumberFromString(asNumber)
	if err != nil {
		return spec, err
	}
	bgpSpec.ASNumber = &num

	ip := d.Get("spec.0.bgp.0.ipv4Address").(string)
	ipV4 := caliconet.IP{net.ParseIP(ip)}
	bgpSpec.IPv4Address = &ipV4

	ip = d.Get("spec.0.bgp.0.ipv6Address").(string)
	ipV6 := caliconet.IP{net.ParseIP(ip)}
	bgpSpec.IPv6Address = &ipV6
	spec.BGP = &bgpSpec

	return spec, nil
}

// set Schema Fields based on existing Node Specs
func setSchemaFieldsForNodeSpec(node *api.Node, d *schema.ResourceData) {
	specArray := make([]interface{}, 1)

	specMap := make(map[string]interface{})

	bgpMapArray := make([]interface{}, 1)

	bgpMap := make(map[string]interface{})

	bgpMap["asNumber"] = node.Spec.BGP.ASNumber
	bgpMap["ipv4Address"] = node.Spec.BGP.IPv4Address
	bgpMap["ipv6Address"] = node.Spec.BGP.IPv6Address
	bgpMapArray[0] = bgpMap

	specMap["bgp"] = bgpMapArray

	specArray[0] = specMap

	d.Set("spec", specArray)
}

func resourceCalicoNodeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	metadata := dToNodeMetadata(d)

	spec, err := dToNodeSpec(d)
	if err != nil {
		return err
	}

	nodes := calicoClient.Nodes()
	if _, err = nodes.Create(&api.Node{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	d.SetId(metadata.Name)
	return resourceCalicoNodeRead(d, meta)
}

func resourceCalicoNodeRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	nodes := calicoClient.Nodes()
	node, err := nodes.Get(api.NodeMetadata{
		Name: d.Get("name").(string),
	})

	// Handle endpoint does not exist
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	d.SetId(d.Get("name").(string))
	setSchemaFieldsForNodeSpec(node, d)

	return nil
}

func resourceCalicoNodeUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	nodes := calicoClient.Nodes()

	// Handle non-existant resource
	metadata := dToNodeMetadata(d)

	if _, err := nodes.Get(metadata); err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	// Simply recreate the complete resource
	spec, err := dToNodeSpec(d)
	if err != nil {
		return err
	}

	if _, err = nodes.Apply(&api.Node{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	return nil
}

func resourceCalicoNodeDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	nodes := calicoClient.Nodes()
	err := nodes.Delete(api.NodeMetadata{
		Name: d.Get("name").(string),
	})

	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	return nil
}
