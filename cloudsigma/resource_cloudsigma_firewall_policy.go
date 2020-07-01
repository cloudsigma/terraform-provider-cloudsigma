package cloudsigma

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaFirewallPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaFirewallPolicyCreate,
		Read:   resourceCloudSigmaFirewallPolicyRead,
		Update: resourceCloudSigmaFirewallPolicyUpdate,
		Delete: resourceCloudSigmaFirewallPolicyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Type: schema.TypeString,
				Required:    true,
			},

			"owner": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_uri": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"uuid": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaFirewallPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[DEBUG] Creating CloudSigma firewall policy...")

	createRequest := &cloudsigma.FirewallPolicyCreateRequest{
		FirewallPolicies: []cloudsigma.FirewallPolicy{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] Firewall policy create configuration: %#v", createRequest)
	firewallPolicies, resp, err := client.FirewallPolicies.Create(context.Background(), createRequest)
	log.Printf("[INFO] response %v", resp)
	if err != nil {
		return fmt.Errorf("error creating firewall policy: %s", err)
	}

	d.SetId(firewallPolicies[0].UUID)
	log.Printf("[INFO] Firewall policy: %s", firewallPolicies[0].UUID)

	return resourceCloudSigmaFirewallPolicyRead(d, meta)
}

func resourceCloudSigmaFirewallPolicyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	firewallPolicy, resp, err := client.FirewallPolicies.Get(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving firewall policy: %s", err)
	}

	_ = d.Set("name", firewallPolicy.Name)
	_ = d.Set("resource_uri", firewallPolicy.ResourceURI)

	owner := []map[string]interface{}{
		{
			"resource_uri": firewallPolicy.Owner.ResourceURI,
			"uuid":         firewallPolicy.Owner.UUID,
		},
	}
	_ = d.Set("owner", owner)

	return nil
}

func resourceCloudSigmaFirewallPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	firewallPolicy := &cloudsigma.FirewallPolicy{
		UUID: d.Id(),
	}

	if name, ok := d.GetOk("name"); ok {
		firewallPolicy.Name = name.(string)
	}

	log.Printf("[DEBUG] Firewall policy update: %#v", firewallPolicy)

	updateRequest := &cloudsigma.FirewallPolicyUpdateRequest{
		FirewallPolicy: firewallPolicy,
	}
	_, _, err := client.FirewallPolicies.Update(context.Background(), firewallPolicy.UUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update firewall policy: %s", err)
	}

	return resourceCloudSigmaFirewallPolicyRead(d, meta)
}

func resourceCloudSigmaFirewallPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[INFO] Deleting firewall policy: %s", d.Id())
	_, err := client.FirewallPolicies.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting firewall policy: %s", err)
	}

	d.SetId("")

	return nil
}
