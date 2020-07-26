package cloudsigma

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaLibraryDrive() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudSigmaLibraryDriveRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"arch": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"media": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"os": {
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

func dataSourceCloudSigmaLibraryDriveRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return fmt.Errorf("issue with filters: %v", filtersOk)
	}

	libdrives, _, err := client.LibraryDrives.List(context.Background())
	if err != nil {
		return fmt.Errorf("error getting libdrives: %v", err)
	}

	libdriveList := make([]cloudsigma.LibraryDrive, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, libdrive := range libdrives {
		sm, err := structToMap(libdrive)
		if err != nil {
			return err
		}

		if filterLoop(f, sm) {
			libdriveList = append(libdriveList, libdrive)
		}
	}

	if len(libdriveList) > 1 {
		return errors.New("your search returned too many results. Please refine your search to be more specific")
	}
	if len(libdriveList) < 1 {
		return errors.New("no results were found")
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
