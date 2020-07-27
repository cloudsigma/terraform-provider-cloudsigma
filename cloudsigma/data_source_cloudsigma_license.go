package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaLicense() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudSigmaLicenseRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"burstable": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"long_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_metric": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaLicenseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	licenses, _, err := client.Licenses.List(ctx)
	if err != nil {
		return diag.Errorf("error getting licenses: %v", err)
	}

	licenseList := make([]cloudsigma.License, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, license := range licenses {
		sm, err := structToMap(license)
		if err != nil {
			return diag.FromErr(err)
		}

		if filterLoop(f, sm) {
			licenseList = append(licenseList, license)
		}
	}

	if len(licenseList) > 1 {
		return diag.FromErr(errors.New("your search returned too many results. Please refine your search to be more specific"))
	}
	if len(licenseList) < 1 {
		return diag.FromErr(errors.New("no results were found"))
	}

	d.SetId(licenseList[0].Name)
	_ = d.Set("burstable", licenseList[0].Burstable)
	_ = d.Set("long_name", licenseList[0].LongName)
	_ = d.Set("name", licenseList[0].Name)
	_ = d.Set("resource_uri", licenseList[0].ResourceURI)
	_ = d.Set("type", licenseList[0].Type)
	_ = d.Set("user_metric", licenseList[0].UserMetric)

	return nil
}
