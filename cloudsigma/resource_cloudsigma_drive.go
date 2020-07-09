package cloudsigma

import (
	"context"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaDrive() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaDriveCreate,
		Read:   resourceCloudSigmaDriveRead,
		Update: resourceCloudSigmaDriveUpdate,
		Delete: resourceCloudSigmaDriveDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"media": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"storage_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaDriveCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	driveCreateRequest := &cloudsigma.DriveCreateRequest{
		Drives: []cloudsigma.Drive{
			{
				Media: d.Get("media").(string),
				Name:  d.Get("name").(string),
				Size:  d.Get("size").(int),
			},
		},
	}
	drives, _, err := client.Drives.Create(context.Background(), driveCreateRequest)
	if err != nil {
		return fmt.Errorf("error creating drive: %s", err)
	}

	d.SetId(drives[0].UUID)

	return resourceCloudSigmaDriveRead(d, meta)
}

func resourceCloudSigmaDriveRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	drive, resp, err := client.Drives.Get(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving drive: %s", err)
	}

	_ = d.Set("media", drive.Media)
	_ = d.Set("name", drive.Name)
	_ = d.Set("resource_uri", drive.ResourceURI)
	_ = d.Set("size", drive.Size)
	_ = d.Set("status", drive.Status)
	_ = d.Set("storage_type", drive.StorageType)

	return nil
}

func resourceCloudSigmaDriveUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	drive := &cloudsigma.Drive{
		UUID: d.Id(),
	}

	if media, ok := d.GetOk("media"); ok {
		drive.Media = media.(string)
	}
	if name, ok := d.GetOk("name"); ok {
		drive.Name = name.(string)
	}
	if size, ok := d.GetOk("size"); ok {
		drive.Size = size.(int)
	}

	updateRequest := &cloudsigma.DriveUpdateRequest{
		Drive: drive,
	}

	_, _, err := client.Drives.Update(context.Background(), drive.UUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update drive: %s", err)
	}

	return resourceCloudSigmaDriveRead(d, meta)
}

func resourceCloudSigmaDriveDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	_, err := client.Drives.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting drive: %s", err)
	}

	d.SetId("")

	return nil
}
