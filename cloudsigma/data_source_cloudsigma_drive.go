package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaDrive() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudSigmaDriveRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaDriveRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	opts := &cloudsigma.DriveListOptions{
		ListOptions: cloudsigma.ListOptions{Limit: 0},
	}
	libdrives, _, err := client.Drives.List(ctx, opts)
	if err != nil {
		return diag.Errorf("error getting drives: %v", err)
	}

	libdriveList := make([]cloudsigma.Drive, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, libdrive := range libdrives {
		sm, err := structToMap(libdrive)
		if err != nil {
			return diag.FromErr(err)
		}

		if filterLoop(f, sm) {
			libdriveList = append(libdriveList, libdrive)
		}
	}

	if len(libdriveList) > 1 {
		return diag.FromErr(errors.New("your search returned too many results. Please refine your search to be more specific"))
	}
	if len(libdriveList) < 1 {
		return diag.FromErr(errors.New("no results were found"))
	}

	d.SetId(libdriveList[0].UUID)
	_ = d.Set("name", libdriveList[0].Name)
	_ = d.Set("size", libdriveList[0].Size)
	_ = d.Set("status", libdriveList[0].Status)
	_ = d.Set("storage_type", libdriveList[0].StorageType)
	_ = d.Set("uuid", libdriveList[0].UUID)

	return nil
}
