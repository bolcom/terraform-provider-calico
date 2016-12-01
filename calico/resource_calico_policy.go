package calico

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/errors"
)

func resourceCalicoPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceCalicoPolicyCreate,
		Read:   resourceCalicoPolicyRead,
		Update: resourceCalicoPolicyUpdate,
		Delete: resourceCalicoPolicyDelete,

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
						"order": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"selector": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
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

func resourceCalicoPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	metadata := dToPolicyMetadata(d)
	spec, err := dToPolicySpec(d)
	if err != nil {
		return err
	}

	policies := calicoClient.Policies()
	if _, err = policies.Create(&api.Policy{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	d.SetId(metadata.Name)
	return resourceCalicoPolicyRead(d, meta)
}

func resourceCalicoPolicyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	policies := calicoClient.Policies()
	policy, err := policies.Get(api.PolicyMetadata{
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

	d.Set("name", policy.Metadata.Name)

	setSchemaFieldsForPolicySpec(policy, d)

	return nil
}

func resourceCalicoPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	policies := calicoClient.Policies()

	metadata := dToPolicyMetadata(d)
	if _, err := policies.Get(metadata); err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			d.SetId("")
			return nil
		}
	}

	spec, err := dToPolicySpec(d)
	if err != nil {
		return err
	}

	if _, err = policies.Apply(&api.Policy{
		Metadata: metadata,
		Spec:     spec,
	}); err != nil {
		return err
	}

	return nil
}

func resourceCalicoPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config)
	calicoClient := config.Client

	policies := calicoClient.Policies()
	err := policies.Delete(api.PolicyMetadata{
		Name: d.Get("name").(string),
	})

	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return fmt.Errorf("ERROR: %v", err)
		}
	}

	return nil
}

// set Schema Fields based on existing Policy Specs
func setSchemaFieldsForPolicySpec(policy *api.Policy, d *schema.ResourceData) {
	specArray := make([]interface{}, 1)

	specMap := make(map[string]interface{})

	specMap["order"] = policy.Spec.Order
	specMap["selector"] = policy.Spec.Selector

	ingressRuleMapArray := make([]interface{}, 1)
	if policy.Spec.IngressRules != nil && len(policy.Spec.IngressRules) > 0 {
		resourceRules := rulesToMap(policy.Spec.IngressRules)
		ruleMap := make(map[string]interface{})
		ruleMap["rule"] = resourceRules
		ingressRuleMapArray[0] = ruleMap
	}
	egressRuleMapArray := make([]interface{}, 1)
	if policy.Spec.EgressRules != nil && len(policy.Spec.EgressRules) > 0 {
		resourceRules := rulesToMap(policy.Spec.EgressRules)
		ruleMap := make(map[string]interface{})
		ruleMap["rule"] = resourceRules
		egressRuleMapArray[0] = ruleMap
	}
	specMap["egress"] = egressRuleMapArray
	specMap["ingress"] = ingressRuleMapArray

	specArray[0] = specMap

	d.Set("spec", specArray)
}

// set Metadata based on existing Policy Metadata
func dToPolicyMetadata(d *schema.ResourceData) api.PolicyMetadata {
	metadata := api.PolicyMetadata{
		Name: d.Get("name").(string),
	}

	return metadata
}

// create Policy based on provided resource data
func dToPolicySpec(d *schema.ResourceData) (api.PolicySpec, error) {
	spec := api.PolicySpec{}

	order := d.Get("spec.0.order").(float64)

	spec.Order = &order

	spec.Selector = d.Get("spec.0.selector").(string)

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
