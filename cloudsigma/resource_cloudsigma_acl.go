package cloudsigma

import (
	"context"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaACL() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudSigmaACLCreate,
		ReadContext:   resourceCloudSigmaACLRead,
		UpdateContext: resourceCloudSigmaACLUpdate,
		DeleteContext: resourceCloudSigmaACLDelete,

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

			// "permissions": {
			// 	Type: schema.TypeSet,
			// 	Elem: &schema.Schema{
			// 		Type: schema.TypeString,
			// 		ValidateFunc: validation.StringInSlice([]string{
			// 			"ATTACH",
			// 			"CLONE",
			// 			"EDIT",
			// 			"LIST",
			// 			"OPEN_VNC",
			// 			"START",
			// 			"STOP",
			// 		}, false),
			// 	},
			// 	Optional: true,
			// },

			"resource_uri": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaACLCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	createRequest := &cloudsigma.ACLCreateRequest{
		ACLs: []cloudsigma.ACL{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] ACL create configuration: %#v", *createRequest)
	acls, _, err := client.ACLs.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(acls[0].UUID)
	log.Printf("[INFO] ACL ID: %s", d.Id())

	return resourceCloudSigmaACLRead(ctx, d, meta)
}

func resourceCloudSigmaACLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	acl, resp, err := client.ACLs.Get(ctx, d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", acl.Name)
	_ = d.Set("resource_uri", acl.ResourceURI)

	// owner := []map[string]interface{}{
	// 	{
	// 		"resource_uri": acl.Owner.ResourceURI,
	// 		"uuid":         acl.Owner.UUID,
	// 	},
	// }
	// _ = d.Set("owner", owner)

	return nil
}

func resourceCloudSigmaACLUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	if d.HasChange("name") {
		_, newName := d.GetChange("name")
		updateRequest := &cloudsigma.ACLUpdateRequest{
			ACL: &cloudsigma.ACL{
				Name: newName.(string),
			},
		}
		log.Printf("[DEBUG] ACL update configuration: %#v", *updateRequest)
		_, _, err := client.ACLs.Update(ctx, d.Id(), updateRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceCloudSigmaACLRead(ctx, d, meta)
}

func resourceCloudSigmaACLDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	_, err := client.ACLs.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
