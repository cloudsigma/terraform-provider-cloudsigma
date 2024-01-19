package cloudsigma

import (
	"encoding/json"
	"log"
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
	for idx := range f {
		if !valuesLoop(f[idx].values, m[f[idx].name]) {
			return false
		}
	}
	return true
}

// valuesLoop search for a matching value for the defined "name".
// Depending on the defined key, it could be a simple string or a JSON object
// The function will search for any matching value.
func valuesLoop(values []string, i interface{}) bool {
	for idx := range values {
		switch intCast := i.(type) {
		case string:
			if values[idx] == i {
				return true
			}
		case map[string]interface{}:
			for idx2 := range intCast {
				if values[idx] == intCast[idx2] {
					return true
				}
			}
		default:
			log.Printf("filter [valuesLoop] wrong type %T", intCast)
		}
	}
	return false
}

func dataSourceFiltersSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"values": {
					Type:     schema.TypeList,
					Required: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}
