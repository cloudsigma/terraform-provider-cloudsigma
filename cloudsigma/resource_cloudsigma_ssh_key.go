package cloudsigma

import (
	"context"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaSSHKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudSigmaSSHKeyCreate,
		ReadContext:   resourceCloudSigmaSSHKeyRead,
		UpdateContext: resourceCloudSigmaSSHKeyUpdate,
		DeleteContext: resourceCloudSigmaSSHKeyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"private_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"public_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	createRequest := &cloudsigma.KeypairCreateRequest{
		Keypairs: []cloudsigma.Keypair{
			{
				Name:      d.Get("name").(string),
				PublicKey: d.Get("public_key").(string),
			},
		},
	}
	log.Printf("[DEBUG] SSH key create configuration: %#v", *createRequest)
	keypairs, _, err := client.Keypairs.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(keypairs[0].UUID)
	log.Printf("[INFO] SSH key ID: %s", d.Id())

	return resourceCloudSigmaSSHKeyRead(ctx, d, meta)
}

func resourceCloudSigmaSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	keypair, resp, err := client.Keypairs.Get(ctx, d.Id())
	if err != nil {
		// If the key is somehow already destroyed, mark as successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", keypair.Name)
	_ = d.Set("private_key", keypair.PrivateKey)
	_ = d.Set("public_key", keypair.PublicKey)

	return nil
}

func resourceCloudSigmaSSHKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	keypair := &cloudsigma.Keypair{
		UUID: d.Id(),
	}

	if name, ok := d.GetOk("name"); ok {
		keypair.Name = name.(string)
	}
	if publicKey, ok := d.GetOk("public_key"); ok {
		keypair.PublicKey = publicKey.(string)
	}

	updateRequest := &cloudsigma.KeypairUpdateRequest{
		Keypair: keypair,
	}
	log.Printf("[DEBUG] SSH key update configuration: %#v", *updateRequest)
	_, _, err := client.Keypairs.Update(context.Background(), d.Id(), updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaSSHKeyRead(ctx, d, meta)
}

func resourceCloudSigmaSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	_, err := client.Keypairs.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
