package cloudsigma

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaSubscription() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudSigmaSubscriptionRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"amount": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_renew": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"free_tier": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"period": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"price": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"remaining": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return fmt.Errorf("issue with filters: %v", filtersOk)
	}

	subscriptions, _, err := client.Subscriptions.List(context.Background())
	if err != nil {
		return fmt.Errorf("error getting subscriptions: %v", err)
	}

	subscriptionList := make([]cloudsigma.Subscription, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, subscription := range subscriptions {
		sm, err := structToMap(subscription)
		if err != nil {
			return err
		}

		if filterLoop(f, sm) {
			subscriptionList = append(subscriptionList, subscription)
		}
	}

	if len(subscriptionList) > 1 {
		return errors.New("your search returned too many results. Please refine your search to be more specific")
	}
	if len(subscriptionList) < 1 {
		return errors.New("no results were found")
	}

	d.SetId(subscriptionList[0].UUID)
	_ = d.Set("amount", subscriptionList[0].Amount)
	_ = d.Set("auto_renew", subscriptionList[0].AutoRenew)
	_ = d.Set("free_tier", subscriptionList[0].FreeTier)
	_ = d.Set("id", subscriptionList[0].ID)
	_ = d.Set("period", subscriptionList[0].Period)
	_ = d.Set("price", subscriptionList[0].Price)
	_ = d.Set("remaining", subscriptionList[0].Remaining)
	_ = d.Set("resource", subscriptionList[0].Resource)
	_ = d.Set("resource_uri", subscriptionList[0].ResourceURI)
	_ = d.Set("status", subscriptionList[0].Status)

	return nil
}
