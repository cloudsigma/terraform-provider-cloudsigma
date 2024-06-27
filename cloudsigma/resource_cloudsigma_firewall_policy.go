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
		Description: `
The firewall policy resource allows you to manage CloudSigma firewall policies.
`,

		CreateContext: resourceCloudSigmaFirewallPolicyCreate,
		ReadContext:   resourceCloudSigmaFirewallPolicyRead,
		UpdateContext: resourceCloudSigmaFirewallPolicyUpdate,
		DeleteContext: resourceCloudSigmaFirewallPolicyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The firewall policy name.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"resource_uri": {
				Description: "The unique resource identifier of the firewall policy.",
				Computed:    true,
				Type:        schema.TypeString,
			},
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
