package cloudsigma

import (
	"context"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaFirewallPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudSigmaFirewallPolicyCreate,
		ReadContext:   resourceCloudSigmaFirewallPolicyRead,
		UpdateContext: resourceCloudSigmaFirewallPolicyUpdate,
		DeleteContext: resourceCloudSigmaFirewallPolicyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			// "owner": {
			// 	Type:     schema.TypeList,
			// 	Optional: true,
			// 	Computed: true,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"resource_uri": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				Computed: true,
			// 			},
			//
			// 			"uuid": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				Computed: true,
			// 			},
			// 		},
			// 	},
			// },

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// "rule": {
			// 	Type:     schema.TypeSet,
			// 	Optional: true,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"action": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				ValidateFunc: validation.StringInSlice([]string{
			// 					"accept",
			// 					"drop",
			// 				}, false),
			// 			},
			// 			"comment": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 			},
			// 			"direction": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 				ValidateFunc: validation.StringInSlice([]string{
			// 					"in",
			// 					"out",
			// 					"both",
			// 				}, false),
			// 			},
			// 			"destination_address": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 			},
			// 			"destination_port_range": {
			// 				Type:         schema.TypeInt,
			// 				Optional:     true,
			// 				ValidateFunc: validation.NoZeroValues,
			// 			},
			// 			"protocol": {
			// 				Type:     schema.TypeString,
			// 				Required: true,
			// 				ValidateFunc: validation.StringInSlice([]string{
			// 					"tcp",
			// 					"udp",
			// 				}, false),
			// 			},
			// 			"source_address": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 			},
			// 			"source_port_range": {
			// 				Type:         schema.TypeInt,
			// 				Optional:     true,
			// 				ValidateFunc: validation.NoZeroValues,
			// 			},
			// 		},
			// 	},
			// },
		},
	}
}

func resourceCloudSigmaFirewallPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	createRequest := &cloudsigma.FirewallPolicyCreateRequest{
		FirewallPolicies: []cloudsigma.FirewallPolicy{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] Firewall policy create configuration: %#v", *createRequest)
	firewallPolicies, _, err := client.FirewallPolicies.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(firewallPolicies[0].UUID)
	log.Printf("[INFO] Firewall policy ID: %s", d.Id())

	return resourceCloudSigmaFirewallPolicyRead(ctx, d, meta)
}

func resourceCloudSigmaFirewallPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	firewallPolicy, resp, err := client.FirewallPolicies.Get(ctx, d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", firewallPolicy.Name)
	_ = d.Set("resource_uri", firewallPolicy.ResourceURI)

	// owner := []map[string]interface{}{
	// 	{
	// 		"resource_uri": firewallPolicy.Owner.ResourceURI,
	// 		"uuid":         firewallPolicy.Owner.UUID,
	// 	},
	// }
	// _ = d.Set("owner", owner)

	return nil
}

func resourceCloudSigmaFirewallPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	firewallPolicy := &cloudsigma.FirewallPolicy{
		UUID: d.Id(),
	}

	if name, ok := d.GetOk("name"); ok {
		firewallPolicy.Name = name.(string)
	}

	log.Printf("[DEBUG] Firewall policy update: %#v", *firewallPolicy)

	updateRequest := &cloudsigma.FirewallPolicyUpdateRequest{
		FirewallPolicy: firewallPolicy,
	}
	_, _, err := client.FirewallPolicies.Update(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaFirewallPolicyRead(ctx, d, meta)
}

func resourceCloudSigmaFirewallPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	_, err := client.FirewallPolicies.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
