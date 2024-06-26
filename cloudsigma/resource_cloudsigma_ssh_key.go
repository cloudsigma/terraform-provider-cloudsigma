package cloudsigma

import (
	"context"
	"log"
	"strings"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaSSHKey() *schema.Resource {
	return &schema.Resource{
		Description: `
The SSH key resource allows you to manage CloudSigma SSH keys.

Keys created with this resource can be referenced in your server
configuration via their IDs.
`,

		CreateContext: resourceCloudSigmaSSHKeyCreate,
		ReadContext:   resourceCloudSigmaSSHKeyRead,
		UpdateContext: resourceCloudSigmaSSHKeyUpdate,
		DeleteContext: resourceCloudSigmaSSHKeyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The SSH key name.",
			},

			"private_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The private SSH key material.",
			},

			"public_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The public SSH key material.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.Trim(old, "\n") == strings.Trim(new, "\n")
				},
			},

			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The SSH key UUID.",
			},
		},
	}
}

func resourceCloudSigmaSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	createRequest := &cloudsigma.KeypairCreateRequest{
		Keypairs: []cloudsigma.Keypair{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	if v, ok := d.GetOk("private_key"); ok {
		createRequest.Keypairs[0].PrivateKey = v.(string)
	}
	if v, ok := d.GetOk("public_key"); ok {
		createRequest.Keypairs[0].PublicKey = strings.Trim(v.(string), "\n")
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
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", keypair.Name)
	_ = d.Set("private_key", keypair.PrivateKey)
	_ = d.Set("public_key", keypair.PublicKey)
	_ = d.Set("uuid", keypair.UUID)

	return nil
}

func resourceCloudSigmaSSHKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	keypair := &cloudsigma.Keypair{
		UUID: d.Id(),
	}

	if d.HasChange("name") {
		keypair.Name = d.Get("name").(string)
	}
	if d.HasChange("private_key") {
		keypair.PrivateKey = d.Get("private_key").(string)
	}
	if d.HasChange("public_key") {
		keypair.PublicKey = d.Get("public_key").(string)
	}

	updateRequest := &cloudsigma.KeypairUpdateRequest{
		Keypair: keypair,
	}
	log.Printf("[DEBUG] SSH key update configuration: %#v", *updateRequest)
	_, _, err := client.Keypairs.Update(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaSSHKeyRead(ctx, d, meta)
}

func resourceCloudSigmaSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	resp, err := client.Keypairs.Delete(ctx, d.Id())
	if err != nil {
		// handle remotely destroyed keys
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
