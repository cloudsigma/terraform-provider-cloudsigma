package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaLibraryDrive() *schema.Resource {
	return &schema.Resource{
		Description: `
The library drive data source provides information about an existing CloudSigma library drive.
`,

		ReadContext: dataSourceCloudSigmaLibraryDriveRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"arch": {
				Description: "The library drive operating system bit architecture.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"description": {
				Description: "The library drive image description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"image_type": {
				Description: "Type of drive image.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"media": {
				Description: "Media representation type. It can be `cdrom` or `disk`.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "Human readable name of the library drive.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"os": {
				Description: "Operating system of the drive.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"size": {
				Description: "Size of the library drive in bytes.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"status": {
				Description: "The library drive status.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"storage_type": {
				Description: "Library drive storage type.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceCloudSigmaLibraryDriveRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	opts := &cloudsigma.LibraryDriveListOptions{
		ListOptions: cloudsigma.ListOptions{Limit: 0},
	}
	libdrives, _, err := client.LibraryDrives.List(ctx, opts)
	if err != nil {
		return diag.Errorf("error getting libdrives: %v", err)
	}

	libdriveList := make([]cloudsigma.LibraryDrive, 0)

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
	_ = d.Set("arch", libdriveList[0].Arch)
	_ = d.Set("description", libdriveList[0].Description)
	_ = d.Set("image_type", libdriveList[0].ImageType)
	_ = d.Set("media", libdriveList[0].Media)
	_ = d.Set("name", libdriveList[0].Name)
	_ = d.Set("os", libdriveList[0].OS)
	_ = d.Set("size", libdriveList[0].Size)
	_ = d.Set("status", libdriveList[0].Status)
	_ = d.Set("storage_type", libdriveList[0].StorageType)

	return nil
}
