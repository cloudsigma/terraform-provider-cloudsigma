package cloudsigma

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceCloudSigmaACL() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaACLCreate,
		Read:   resourceCloudSigmaACLRead,
		Update: resourceCloudSigmaACLUpdate,
		Delete: resourceCloudSigmaACLDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
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

			"permissions": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"ATTACH",
						"CLONE",
						"EDIT",
						"LIST",
						"OPEN_VNC",
						"START",
						"STOP",
					}, false),
				},
				Optional: true,
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaACLCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[DEBUG] Creating CloudSigma ACL...")

	createRequest := &cloudsigma.ACLCreateRequest{
		ACLs: []cloudsigma.ACL{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] ACL create configuration: %#v", createRequest)
	acls, _, err := client.ACLs.Create(context.Background(), createRequest)
	if err != nil {
		return fmt.Errorf("error creating ACL: %s", err)
	}

	d.SetId(acls[0].UUID)
	log.Printf("[INFO] ACL: %s", acls[0].UUID)

	return resourceCloudSigmaACLRead(d, meta)
}

func resourceCloudSigmaACLRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	acl, resp, err := client.ACLs.Get(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving acl: %s", err)
	}

	_ = d.Set("name", acl.Name)
	_ = d.Set("resource_uri", acl.ResourceURI)

	owner := []map[string]interface{}{
		{
			"resource_uri": acl.Owner.ResourceURI,
			"uuid":         acl.Owner.UUID,
		},
	}
	_ = d.Set("owner", owner)

	return nil
}

func resourceCloudSigmaACLUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	acl := &cloudsigma.ACL{
		UUID: d.Id(),
	}

	if name, ok := d.GetOk("name"); ok {
		acl.Name = name.(string)
	}

	log.Printf("[DEBUG] ACL update: %#v", acl)

	updateRequest := &cloudsigma.ACLUpdateRequest{
		ACL: acl,
	}
	_, _, err := client.ACLs.Update(context.Background(), acl.UUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update ACL: %s", err)
	}

	return resourceCloudSigmaACLRead(d, meta)
}

func resourceCloudSigmaACLDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[INFO] Deleting ACL: %s", d.Id())
	_, err := client.ACLs.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting ACL: %s", err)
	}

	d.SetId("")

	return nil
}
