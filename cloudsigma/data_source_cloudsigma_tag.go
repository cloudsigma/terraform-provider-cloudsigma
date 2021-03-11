package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaTag() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudSigmaTagRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaTagRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	tags, _, err := client.Tags.List(ctx)
	if err != nil {
		return diag.Errorf("error getting tags: %v", err)
	}

	tagList := make([]cloudsigma.Tag, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, tag := range tags {
		sm, err := structToMap(tag)
		if err != nil {
			return diag.FromErr(err)
		}

		if filterLoop(f, sm) {
			tagList = append(tagList, tag)
		}
	}

	if len(tagList) > 1 {
		return diag.FromErr(errors.New("your search returned too many results. Please refine your search to be more specific"))
	}
	if len(tagList) < 1 {
		return diag.FromErr(errors.New("no results were found"))
	}

	d.SetId(tagList[0].UUID)
	_ = d.Set("name", tagList[0].Name)
	_ = d.Set("resource_uri", tagList[0].ResourceURI)
	_ = d.Set("uuid", tagList[0].UUID)

	return nil
}
