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
			"guest": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"host_availability_zones": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"remote_snapshots": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaCapabilitiesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	cloudStatus, _, err := client.CloudStatus.Get(context.Background())
	if err != nil {
		return fmt.Errorf("error getting cloud status: %v", err)
	}

	d.SetId(hashcode.Strings(cloudStatus.SSO))
	_ = d.Set("guest", cloudStatus.Guest)
	_ = d.Set("host_availability_zones", cloudStatus.HostAvailabilityZones)
	_ = d.Set("remote_snapshots", cloudStatus.RemoteSnapshots)

	return nil
}
