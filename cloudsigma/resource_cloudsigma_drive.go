package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceCloudSigmaDrive() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaDriveCreate,
		Read:   resourceCloudSigmaDriveRead,
		Update: resourceCloudSigmaDriveUpdate,
		Delete: resourceCloudSigmaDriveDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"clone_drive_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"media": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"cdrom", "disk"}, false),
			},

			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"size": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(536870912), // 536870912 = 512MB
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"storage_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaDriveCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)
	ctx := context.Background()

	cloneDriveUUID := d.Get("clone_drive_id").(string)
	if cloneDriveUUID != "" {
		// Clone the Drive if 'clone_drive_id' is set
		cloneRequest := &cloudsigma.DriveCloneRequest{
			Drive: &cloudsigma.Drive{
				Media: d.Get("media").(string),
				Name:  d.Get("name").(string),
				Size:  d.Get("size").(int),
			},
		}
		log.Printf("[DEBUG] Drive clone configuration: %#+v", cloneRequest)
		drive, _, err := client.Drives.Clone(ctx, cloneDriveUUID, cloneRequest)
		if err != nil {
			return fmt.Errorf("error cloning drive: %s", err)
		}

		d.SetId(drive.UUID)
		log.Printf("[INFO] Drive ID: %s", d.Id())
	} else {
		// Create the Drive because 'clone_drive_id' is not set
		createRequest := &cloudsigma.DriveCreateRequest{
			Drives: []cloudsigma.Drive{
				{
					Media: d.Get("media").(string),
					Name:  d.Get("name").(string),
					Size:  d.Get("size").(int),
				},
			},
		}
		log.Printf("[DEBUG] Drive create configuration: %#v", *createRequest)
		drives, _, err := client.Drives.Create(ctx, createRequest)
		if err != nil {
			return fmt.Errorf("error creating drive: %s", err)
		}
		drive := &drives[0]

		d.SetId(drive.UUID)
		log.Printf("[INFO] Drive ID: %s", d.Id())
	}

	log.Printf("[DEBUG] Waiting for Drive (%s) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"cloning_dst", "creating"},
		Target:     []string{"unmounted"},
		Refresh:    driveStateRefreshFunc(client, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("error waiting for Drive (%s) to become available: %s", d.Id(), err)
	}

	return resourceCloudSigmaDriveRead(d, meta)
}

func resourceCloudSigmaDriveRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	// Refresh the Drive state
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

func driveStateRefreshFunc(client *cloudsigma.Client, uuid string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		drive, _, err := client.Drives.Get(context.Background(), uuid)
		if err != nil {
			return nil, "", fmt.Errorf("error retrieving drive with uuid %s: %s", uuid, err)
		}

		return drive, drive.Status, nil
	}
}
