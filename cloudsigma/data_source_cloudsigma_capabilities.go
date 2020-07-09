package cloudsigma

import (
	"context"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceCloudSigmaCapabilities() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudSigmaCapabilitiesRead,

		Schema: map[string]*schema.Schema{
			"hypervisors": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kvm": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func dataSourceCloudSigmaCapabilitiesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	capabilities, _, err := client.Capabilities.Get(context.Background())
	if err != nil {
		return fmt.Errorf("error getting capabilities: %v", err)
	}

	d.SetId(hashcode.Strings(capabilities.Hypervisors.KVM))

	kvm := []map[string]interface{}{
		{
			"kvm": capabilities.Hypervisors.KVM,
		},
	}
	_ = d.Set("kvm", kvm)

	return nil
}
