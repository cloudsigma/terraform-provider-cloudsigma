package cloudsigma

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaSSHKeyCreate,
		Read:   resourceCloudSigmaSSHKeyRead,
		Update: resourceCloudSigmaSSHKeyUpdate,
		Delete: resourceCloudSigmaSSHKeyDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"private_key": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"public_key": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceCloudSigmaSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	keypairCreateRequest := &cloudsigma.KeypairCreateRequest{
		Keypairs: []cloudsigma.Keypair{
			{
				Name:      d.Get("name").(string),
				PublicKey: d.Get("public_key").(string),
			},
		},
	}

	log.Printf("[DEBUG] SSH key create configuration: %#v", keypairCreateRequest)
	keypairs, _, err := client.Keypairs.Create(context.Background(), keypairCreateRequest)
	if err != nil {
		return fmt.Errorf("error creating SSH key: %s", err)
	}

	keypair := keypairs[0]

	d.SetId(keypair.UUID)
	log.Printf("[INFO] SSH key: %s", keypair.UUID)

	return resourceCloudSigmaSSHKeyRead(d, meta)
}

func resourceCloudSigmaSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	keypair, resp, err := client.Keypairs.Get(context.Background(), d.Id())
	if err != nil {
		// If the key is somehow already destroyed, mark as successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving SSH key: %s", err)
	}

	_ = d.Set("name", keypair.Name)
	_ = d.Set("private_key", keypair.PrivateKey)
	_ = d.Set("public_key", keypair.PublicKey)

	return nil
}

func resourceCloudSigmaSSHKeyUpdate(d *schema.ResourceData, meta interface{}) error {
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

	log.Printf("[DEBUG] SSH key update: %#v", keypair)

	updateRequest := &cloudsigma.KeypairUpdateRequest{
		Keypair: keypair,
	}
	_, _, err := client.Keypairs.Update(context.Background(), keypair.UUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update SSH key: %s", err)
	}

	return resourceCloudSigmaSSHKeyRead(d, meta)
}

func resourceCloudSigmaSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[INFO] Deleting SSH key: %s", d.Id())
	_, err := client.Keypairs.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting SSH key: %s", err)
	}

	d.SetId("")

	return nil
}
