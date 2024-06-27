package cloudsigma

import (
	"context"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaTag() *schema.Resource {
	return &schema.Resource{
		Description: `
The tag resource allows you to manage CloudSigma tags.

A tag is a label that can be applied to a CloudSigma resource in order to better organize or
facilitate the lookups and actions on it. Tags created with this resource can be referenced
in your configurations via their IDs.
`,

		CreateContext: resourceCloudSigmaTagCreate,
		ReadContext:   resourceCloudSigmaTagRead,
		UpdateContext: resourceCloudSigmaTagUpdate,
		DeleteContext: resourceCloudSigmaTagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The tag name.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"resource_uri": {
				Description: "The unique resource identifier of the tag.",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceCloudSigmaTagCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	// build create configuration
	createRequest := &cloudsigma.TagCreateRequest{
		Tags: []cloudsigma.Tag{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] Tag create configuration: %#v", *createRequest)
	tags, _, err := client.Tags.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tags[0].UUID)
	log.Printf("[INFO] Tag ID: %s", d.Id())

	return resourceCloudSigmaTagRead(ctx, d, meta)
}

func resourceCloudSigmaTagRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	tag, resp, err := client.Tags.Get(ctx, d.Id())
	if err != nil {
		// If the tag is somehow already destroyed, mark as successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", tag.Name)
	_ = d.Set("resource_uri", tag.ResourceURI)

	// owner := []map[string]interface{}{
	// 	{
	// 		"resource_uri": tag.Owner.ResourceURI,
	// 		"uuid":         tag.Owner.UUID,
	// 	},
	// }
	// _ = d.Set("owner", owner)

	return nil
}

func resourceCloudSigmaTagUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	if d.HasChange("name") {
		_, newName := d.GetChange("name")
		updateRequest := &cloudsigma.TagUpdateRequest{
			Tag: &cloudsigma.Tag{
				Name: newName.(string),
			},
		}
		log.Printf("[DEBUG] Tag update configuration: %#v", *updateRequest)
		_, _, err := client.Tags.Update(context.Background(), d.Id(), updateRequest)
		if err != nil {
			return diag.FromErr(err)
		}

	}

	return resourceCloudSigmaTagRead(ctx, d, meta)
}

func resourceCloudSigmaTagDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	_, err := client.Tags.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
