package cloudsigma

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type filter struct {
	name   string
	values []string
}

func buildCloudSigmaDataSourceFilter(set *schema.Set) []filter {
	var filters []filter

	for _, v := range set.List() {
		m := v.(map[string]interface{})
		var values []string
		for _, value := range m["values"].([]interface{}) {
			values = append(values, value.(string))
		}
		filters = append(filters, filter{
			name:   m["name"].(string),
			values: values,
		})
	}

	return filters
}

func structToMap(data interface{}) (map[string]interface{}, error) {
	var structMap map[string]interface{}

	a, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(a, &structMap)
	if err != nil {
		return nil, err
	}

	newMap := make(map[string]interface{})
	for k, v := range structMap {
		switch v := v.(type) {
		case string:
			newMap[strings.ToLower(k)] = v
		case bool:
			newMap[strings.ToLower(k)] = strconv.FormatBool(v)
		case int:
			newMap[strings.ToLower(k)] = strconv.FormatInt(int64(v), 10)
		case float64:
			newMap[strings.ToLower(k)] = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			newMap[strings.ToLower(k)] = v
		}
	}

	return newMap, nil
}

func filterLoop(f []filter, m map[string]interface{}) bool {
	for _, filter := range f {
		if !valuesLoop(filter.values, m[filter.name]) {
			return false
		}
	}
	return true
}

func valuesLoop(values []string, i interface{}) bool {
	for _, v := range values {
		if v == i {
			return true
		}
	}
	return false
}

func dataSourceFiltersSchema() *schema.Schema {
	return &schema.Schema{
		Description: "One or more name/value pairs to filter off of.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Description: "The name of the attribute to filter.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"values": {
					Description: "The value of the attribute to filter.",
					Type:        schema.TypeList,
					Required:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}
