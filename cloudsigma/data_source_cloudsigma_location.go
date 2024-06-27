package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaLocation() *schema.Resource {
	return &schema.Resource{
		Description: `
The location data source provides information about an existing CloudSigma location.
`,

		ReadContext: dataSourceCloudSigmaLocationRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"api_endpoint": {
				Description: "The API endpoint of the location.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"country_code": {
				Description: "The location country code.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"display_name": {
				Description: "Human readable location name.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"id": {
				Description: "The location ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceCloudSigmaLocationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	locations, _, err := client.Locations.List(context.Background())
	if err != nil {
		return diag.Errorf("error getting locations: %v", err)
	}

	locationList := make([]cloudsigma.Location, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, location := range locations {
		sm, err := structToMap(location)
		if err != nil {
			return diag.FromErr(err)
		}

		if filterLoop(f, sm) {
			locationList = append(locationList, location)
		}
	}

	if len(locationList) > 1 {
		return diag.FromErr(errors.New("your search returned too many results. Please refine your search to be more specific"))
	}
	if len(locationList) < 1 {
		return diag.FromErr(errors.New("no results were found"))
	}

	d.SetId(locationList[0].ID)
	_ = d.Set("api_endpoint", locationList[0].APIEndpoint)
	_ = d.Set("country_code", locationList[0].CountryCode)
	_ = d.Set("display_name", locationList[0].DisplayName)
	_ = d.Set("id", locationList[0].ID)

	return nil
}
