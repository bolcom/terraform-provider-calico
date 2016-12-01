package calico

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/projectcalico/libcalico-go/lib/api"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

// read existing Source/Destination Entity Rules into a map for easy consumption
func getEntityRuleMap(entityRule api.EntityRule) map[string]interface{} {
	resourceSourceMap := make(map[string]interface{})

	if entityRule.Net != nil {
		resourceSourceMap["net"] = entityRule.Net.String()
	}
	if len(entityRule.Selector) > 0 {
		resourceSourceMap["selector"] = entityRule.Selector
	}
	if len(entityRule.Ports) > 0 {
		portsArray := make([]string, len(entityRule.Ports))
		for i, v := range entityRule.Ports {
			val := v.String()
			portsArray[i] = val
		}
		resourceSourceMap["ports"] = portsArray
	}
	if entityRule.NotNet != nil {
		resourceSourceMap["notNet"] = entityRule.NotNet.String()
	}
	if len(entityRule.NotSelector) > 0 {
		resourceSourceMap["notSelector"] = entityRule.NotSelector
	}
	if len(entityRule.NotPorts) > 0 {
		notPortsArray := make([]string, len(entityRule.NotPorts))
		for i, v := range entityRule.NotPorts {
			val := v.String()
			notPortsArray[i] = val
		}
		resourceSourceMap["notPorts"] = notPortsArray
	}
	return resourceSourceMap
}

// convert resourceMap to an api.Rule
func resourceMapToRule(mapStruct map[string]interface{}) (api.Rule, error) {
	rule := api.Rule{}

	if val, ok := mapStruct["action"]; ok {
		rule.Action = val.(string)
	}
	if val, ok := mapStruct["protocol"]; ok {
		if len(val.(string)) > 0 {
			protocol := numorstring.ProtocolFromString(val.(string))
			rule.Protocol = &protocol
		}
	}
	if val, ok := mapStruct["notProtocol"]; ok {
		if len(val.(string)) > 0 {
			notProtocol := numorstring.ProtocolFromString(val.(string))
			rule.NotProtocol = &notProtocol
		}
	}
	if val, ok := mapStruct["icmp"]; ok {
		icmpList := val.([]interface{})
		if len(icmpList) > 0 {
			for _, v := range icmpList {
				icmpMap := v.(map[string]interface{})
				icmpType := icmpMap["type"].(int)
				icmpCode := icmpMap["code"].(int)
				icmp := api.ICMPFields{
					Type: &icmpType,
					Code: &icmpCode,
				}
				rule.ICMP = &icmp
			}
		}
	}
	if val, ok := mapStruct["notICMP"]; ok {
		icmpList := val.([]interface{})
		if len(icmpList) > 0 {
			for _, v := range icmpList {
				icmpMap := v.(map[string]interface{})
				icmpType := icmpMap["type"].(int)
				icmpCode := icmpMap["code"].(int)
				icmp := api.ICMPFields{
					Type: &icmpType,
					Code: &icmpCode,
				}
				rule.NotICMP = &icmp
			}
		}
	}
	if val, ok := mapStruct["source"]; ok {
		sourceList := val.([]interface{})

		if len(sourceList) > 0 {
			srcEntityRules, err := srcDstListToEntityRule(sourceList)
			if err != nil {
				return rule, err
			}
			rule.Source = srcEntityRules
		}
	}

	if val, ok := mapStruct["destination"]; ok {
		destinationList := val.([]interface{})

		if len(destinationList) > 0 {
			destEntityRules, err := srcDstListToEntityRule(destinationList)
			if err != nil {
				return rule, err
			}
			rule.Destination = destEntityRules
		}
	}

	return rule, nil
}

// convert resource destination/source structs to a api.EntityRule
func srcDstListToEntityRule(srcDstList []interface{}) (api.EntityRule, error) {
	entityRule := api.EntityRule{}
	resourceRuleMap := srcDstList[0].(map[string]interface{})

	if v, ok := resourceRuleMap["net"]; ok {
		if len(v.(string)) > 0 {
			_, n, err := caliconet.ParseCIDR(v.(string))
			if err != nil {
				return entityRule, err
			}
			entityRule.Net = n
		}
	}
	if v, ok := resourceRuleMap["selector"]; ok {
		entityRule.Selector = v.(string)
	}
	if v, ok := resourceRuleMap["notSelector"]; ok {
		entityRule.NotSelector = v.(string)
	}
	if v, ok := resourceRuleMap["ports"]; ok {
		if resourcePortList, ok := v.([]interface{}); ok {
			portList, err := toPortList(resourcePortList)
			if err != nil {
				return entityRule, err
			}
			entityRule.Ports = portList
		}
	}
	if v, ok := resourceRuleMap["notPorts"]; ok {
		if resourcePortList, ok := v.([]interface{}); ok {
			portList, err := toPortList(resourcePortList)
			if err != nil {
				return entityRule, err
			}
			entityRule.NotPorts = portList
		}
	}
	return entityRule, nil
}

// create an array of Ports
func toPortList(resourcePortList []interface{}) ([]numorstring.Port, error) {
	portList := make([]numorstring.Port, len(resourcePortList))

	for i, v := range resourcePortList {
		p, err := numorstring.PortFromString(v.(string))
		if err != nil {
			return portList, err
		}
		portList[i] = p
	}
	return portList, nil
}

// check if Entity Rule is empty
func nonEmptyEntityRule(entityRule *api.EntityRule) bool {
	state := false

	if len(entityRule.Tag) > 0 {
		state = true
	}
	if entityRule.Net != nil {
		state = true
	}
	if len(entityRule.Selector) > 0 {
		state = true
	}
	if len(entityRule.Ports) > 0 {
		state = true
	}
	if len(entityRule.NotTag) > 0 {
		state = true
	}
	if entityRule.NotNet != nil {
		state = true
	}
	if len(entityRule.NotSelector) > 0 {
		state = true
	}
	if len(entityRule.NotPorts) > 0 {
		state = true
	}

	return state
}

// read []api.Rules into a map for easy consumption
func rulesToMap(calicoRules []api.Rule) []map[string]interface{} {
	resourceRules := make([]map[string]interface{}, len(calicoRules))

	for i, rule := range calicoRules {
		resourceRule := make(map[string]interface{})
		if len(rule.Action) > 0 {
			resourceRule["action"] = rule.Action
		}
		if rule.Protocol != nil {
			resourceRule["protocol"] = rule.Protocol.String()
		}
		if rule.ICMP != nil {
			resourceIcmpArray := make([]map[string]int, 1)
			resourceIcmpMap := make(map[string]int)

			resourceIcmpMap["code"] = *rule.ICMP.Code
			resourceIcmpMap["type"] = *rule.ICMP.Type

			resourceIcmpArray[0] = resourceIcmpMap
			resourceRule["icmp"] = resourceIcmpArray
		}
		if rule.NotICMP != nil {
			resourceIcmpArray := make([]map[string]int, 1)
			resourceIcmpMap := make(map[string]int)

			resourceIcmpMap["code"] = *rule.ICMP.Code
			resourceIcmpMap["type"] = *rule.ICMP.Type

			resourceIcmpArray[0] = resourceIcmpMap
			resourceRule["notICMP"] = resourceIcmpArray
		}
		if nonEmptyEntityRule(&rule.Source) {
			resourceSourceArray := make([]map[string]interface{}, 1)

			resourceSourceArray[0] = getEntityRuleMap(rule.Source)
			resourceRule["source"] = resourceSourceArray
		}
		if nonEmptyEntityRule(&rule.Destination) {
			resourceSourceArray := make([]map[string]interface{}, 1)

			resourceSourceArray[0] = getEntityRuleMap(rule.Destination)
			resourceRule["destination"] = resourceSourceArray
		}
		resourceRules[i] = resourceRule
	}

	return resourceRules

}

func dToCIDR(d *schema.ResourceData, field string) (caliconet.IPNet, error) {
	_, cidr, err := caliconet.ParseCIDR(d.Get(field).(string))
	if err != nil {
		return *cidr, fmt.Errorf("ERROR: couldn't parse CIDR: %v", err)
	}
	return *cidr, nil
}

func entityRuleSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"net": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"notNet": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"selector": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"notSelector": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ports": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"notPorts": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func ruleSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"rule": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"notProtocol": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
									},
									"code": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},
						"notICMP": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
									},
									"code": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},
						"source": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     entityRuleSchema(),
						},
						"destination": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     entityRuleSchema(),
						},
					},
				},
			},
		},
	}
}
