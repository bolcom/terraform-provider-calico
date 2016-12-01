package calico

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/errors"
)

func resourceCalicoProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoProfileCreate,
		Read:   resourceCalicoProfileRead,
		Update: resourceCalicoProfileUpdate,
		Delete: resourceCalicoProfileDelete,

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
			"spec": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ingress": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     ruleSchema(),
						},
						"egress": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     ruleSchema(),
						},
					},
				},
			},
		},
	}
}

func resourceCalicoProfileCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	metadata := dToProfileMetadata(d)
	spec, err := dToProfileSpec(d)
	if err != nil {
		return err
	}

	profiles := calicoClient.Profiles()
	if _, err = profiles.Create(&api.Profile{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	d.SetId(metadata.Name)
	return resourceCalicoProfileRead(d, meta)
}

func resourceCalicoProfileRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	profiles := calicoClient.Profiles()
	profile, err := profiles.Get(api.ProfileMetadata{
		Name: d.Get("name").(string),
	})

	// Deal with resource does not exist
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		} else {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	d.Set("name", profile.Metadata.Name)
	d.Set("labels", profile.Metadata.Labels)

	setSchemaFieldsForProfileSpec(profile, d)

	return nil
}

func resourceCalicoProfileUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	profiles := calicoClient.Profiles()

	metadata := dToProfileMetadata(d)
	if _, err := profiles.Get(metadata); err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	spec, err := dToProfileSpec(d)
	if err != nil {
		return err
	}

	if _, err = profiles.Apply(&api.Profile{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	return nil
}

func resourceCalicoProfileDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	profiles := calicoClient.Profiles()
	err := profiles.Delete(api.ProfileMetadata{
		Name: d.Get("name").(string),
	})

	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	return nil
}

// set Schema Fields based on existing Profile Specs
func setSchemaFieldsForProfileSpec(profile *api.Profile, d *schema.ResourceData) {
	specArray := make([]interface{}, 1)

	specMap := make(map[string]interface{})
	ingressRuleMapArray := make([]interface{}, 1)
	if profile.Spec.IngressRules != nil && len(profile.Spec.IngressRules) > 0 {
		resourceRules := rulesToMap(profile.Spec.IngressRules)
		ruleMap := make(map[string]interface{})
		ruleMap["rule"] = resourceRules
		ingressRuleMapArray[0] = ruleMap
	}
	egressRuleMapArray := make([]interface{}, 1)
	if profile.Spec.EgressRules != nil && len(profile.Spec.EgressRules) > 0 {
		resourceRules := rulesToMap(profile.Spec.EgressRules)
		ruleMap := make(map[string]interface{})
		ruleMap["rule"] = resourceRules
		egressRuleMapArray[0] = ruleMap
	}
	specMap["egress"] = egressRuleMapArray
	specMap["ingress"] = ingressRuleMapArray

	specArray[0] = specMap

	d.Set("spec", specArray)
}

// set Metadata based on existing Profile Metadata
func dToProfileMetadata(d *schema.ResourceData) api.ProfileMetadata {
	metadata := api.ProfileMetadata{
		Name: d.Get("name").(string),
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

// create Profile based on provided resource data
func dToProfileSpec(d *schema.ResourceData) (api.ProfileSpec, error) {
	spec := api.ProfileSpec{}

	if v, ok := d.GetOk("spec.0.ingress.0.rule.#"); ok {
		ingressRules := make([]api.Rule, v.(int))

		for i := range ingressRules {
			mapStruct := d.Get("spec.0.ingress.0.rule." + strconv.Itoa(i)).(map[string]interface{})

			rule, err := resourceMapToRule(mapStruct)
			if err != nil {
				return spec, err
			}

			ingressRules[i] = rule
		}
		spec.IngressRules = ingressRules
	}
	if v, ok := d.GetOk("spec.0.egress.0.rule.#"); ok {
		egressRules := make([]api.Rule, v.(int))

		for i := range egressRules {
			mapStruct := d.Get("spec.0.egress.0.rule." + strconv.Itoa(i)).(map[string]interface{})

			rule, err := resourceMapToRule(mapStruct)
			if err != nil {
				return spec, err
			}

			egressRules[i] = rule
		}
		spec.EgressRules = egressRules
	}

	return spec, nil
}
