package calico

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/errors"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/scope"
)

func resourceCalicoBgpPeer() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoBgpPeerCreate,
		Read:   resourceCalicoBgpPeerRead,
		Update: resourceCalicoBgpPeerUpdate,
		Delete: resourceCalicoBgpPeerDelete,

		Schema: map[string]*schema.Schema{
			"scope": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"node": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"peerIP": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"spec": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"asNumber": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func dToBgpPeerMetadata(d *schema.ResourceData) (api.BGPPeerMetadata, error) {

	metadata := api.BGPPeerMetadata{
		Node: d.Get("node").(string),
	}

	pIP := d.Get("peerIP").(string)
	peerIP := caliconet.IP{net.ParseIP(pIP)}
	metadata.PeerIP = peerIP

	metadata.Scope = scope.Scope(d.Get("scope").(string))

	return metadata, nil
}

func dToBgpPeerSpec(d *schema.ResourceData) (api.BGPPeerSpec, error) {
	spec := api.BGPPeerSpec{}

	asNumber := d.Get("spec.0.asNumber").(string)
	if num, err := numorstring.ASNumberFromString(asNumber); err != nil {
		return spec, err
	} else {
		spec.ASNumber = num
	}

	return spec, nil
}

// set Schema Fields based on existing BGPPeer Specs
func setSchemaFieldsForBGPPeerSpec(bgpPeer *api.BGPPeer, d *schema.ResourceData) {
	specArray := make([]interface{}, 1)

	specMap := make(map[string]interface{})

	specMap["asNumber"] = bgpPeer.Spec.ASNumber.String()
	specArray[0] = specMap

	d.Set("spec", specArray)
}

func resourceCalicoBgpPeerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	metadata, err := dToBgpPeerMetadata(d)
	if err != nil {
		return err
	}
	spec, err := dToBgpPeerSpec(d)
	if err != nil {
		return err
	}

	bgpPeers := calicoClient.BGPPeers()
	if _, err = bgpPeers.Create(&api.BGPPeer{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	compoundID := string(metadata.Scope) + "_" + metadata.Node + "_" + metadata.PeerIP.String()
	d.SetId(compoundID)
	return resourceCalicoBgpPeerRead(d, meta)
}

func resourceCalicoBgpPeerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	bgpPeers := calicoClient.BGPPeers()

	ip := d.Get("peerIP").(string)
	resourcePeerIP := caliconet.IP{net.ParseIP(ip)}

	resourceNode := d.Get("node").(string)
	resourceScope := scope.Scope(d.Get("scope").(string))

	bgpPeer, err := bgpPeers.Get(api.BGPPeerMetadata{
		Scope:  resourceScope,
		Node:   resourceNode,
		PeerIP: resourcePeerIP,
	})

	// Handle endpoint does not exist
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	compoundID := d.Get("scope").(string) + "_" + d.Get("node").(string) + "_" + d.Get("peerIP").(string)
	d.SetId(compoundID)

	setSchemaFieldsForBGPPeerSpec(bgpPeer, d)

	return nil
}

func resourceCalicoBgpPeerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	bgpPeers := calicoClient.BGPPeers()

	// Handle non-existant resource
	metadata, err := dToBgpPeerMetadata(d)
	if err != nil {
		return err
	}
	if _, err := bgpPeers.Get(metadata); err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	// Simply recreate the complete resource
	spec, err := dToBgpPeerSpec(d)
	if err != nil {
		return err
	}

	if _, err = bgpPeers.Apply(&api.BGPPeer{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	return nil
}

func resourceCalicoBgpPeerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	bgpPeers := calicoClient.BGPPeers()

	ip := d.Get("peerIP").(string)
	resourcePeerIP := caliconet.IP{net.ParseIP(ip)}

	resourceNode := d.Get("node").(string)
	resourceScope := scope.Scope(d.Get("scope").(string))

	err := bgpPeers.Delete(api.BGPPeerMetadata{
		Scope:  resourceScope,
		Node:   resourceNode,
		PeerIP: resourcePeerIP,
	})

	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	return nil
}
