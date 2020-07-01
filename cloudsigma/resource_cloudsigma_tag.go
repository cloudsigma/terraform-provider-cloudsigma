package cloudsigma

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaTag() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaTagCreate,
		Read:   resourceCloudSigmaTagRead,
		Update: resourceCloudSigmaTagUpdate,
		Delete: resourceCloudSigmaTagDelete,

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

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaTagCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[DEBUG] Creating CloudSigma tag...")

	tagCreateRequest := &cloudsigma.TagCreateRequest{
		Tags: []cloudsigma.Tag{
			{
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] Tag create configuration: %#v", tagCreateRequest)
	tags, _, err := client.Tags.Create(context.Background(), tagCreateRequest)
	if err != nil {
		return fmt.Errorf("error creating tag: %s", err)
	}

	d.SetId(tags[0].UUID)
	log.Printf("[INFO] Tag: %s", tags[0].UUID)

	return resourceCloudSigmaTagRead(d, meta)
}

func resourceCloudSigmaTagRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	tag, resp, err := client.Tags.Get(context.Background(), d.Id())
	if err != nil {
		// If the tag is somehow already destroyed, mark as successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving tag: %s", err)
	}

	_ = d.Set("name", tag.Name)
	_ = d.Set("resource_uri", tag.ResourceURI)

	owner := []map[string]interface{}{
		{
			"resource_uri": tag.Owner.ResourceURI,
			"uuid":         tag.Owner.UUID,
		},
	}
	_ = d.Set("owner", owner)

	return nil
}

func resourceCloudSigmaTagUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	tag := &cloudsigma.Tag{
		UUID: d.Id(),
	}

	if name, ok := d.GetOk("name"); ok {
		tag.Name = name.(string)
	}

	log.Printf("[DEBUG] Tag update: %#v", tag)

	updateRequest := &cloudsigma.TagUpdateRequest{
		Name: tag.Name,
	}
	_, _, err := client.Tags.Update(context.Background(), tag.UUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update tag: %s", err)
	}

	return resourceCloudSigmaTagRead(d, meta)
}

func resourceCloudSigmaTagDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	log.Printf("[INFO] Deleting tag: %s", d.Id())
	_, err := client.Tags.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting tag: %s", err)
	}

	d.SetId("")

	return nil
}
