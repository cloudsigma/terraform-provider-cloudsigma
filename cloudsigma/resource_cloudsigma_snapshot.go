package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaSnapshotCreate,
		Read:   resourceCloudSigmaSnapshotRead,
		Update: resourceCloudSigmaSnapshotUpdate,
		Delete: resourceCloudSigmaSnapshotDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"drive": {
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

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudSigmaSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	createRequest := &cloudsigma.SnapshotCreateRequest{
		Snapshots: []cloudsigma.Snapshot{
			{
				Drive: cloudsigma.Drive{
					UUID: d.Get("drive").(string),
				},
				Name: d.Get("name").(string),
			},
		},
	}

	// check if drive status is mounted or unmounted
	driveUUID := d.Get("drive").(string)
	retryCount := 5
	currentRetry := 1
	for {
		drive, _, err := client.Drives.Get(context.Background(), driveUUID)
		if err != nil {
			log.Printf("[DEBUG] error getting drive with uuid: %v", driveUUID)
			if currentRetry <= retryCount {
				currentRetry++
				log.Printf("[DEBUG] waiting 5 seconds before next call...")
				time.Sleep(5 * time.Second)
				continue
			}
			return fmt.Errorf("error getting drive with uuid %v: %v", driveUUID, err)
		}
		if drive.Status == "mounted" || drive.Status == "unmounted" {
			break
		}
		time.Sleep(2 * time.Second)
	}

	snapshots, _, err := client.Snapshots.Create(context.Background(), createRequest)
	if err != nil {
		return fmt.Errorf("error creating snapshot: %s", err)
	}

	d.SetId(snapshots[0].UUID)

	return resourceCloudSigmaSnapshotRead(d, meta)
}

func resourceCloudSigmaSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	snapshot, resp, err := client.Snapshots.Get(context.Background(), d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving snapshot: %s", err)
	}

	_ = d.Set("drive", snapshot.Drive.UUID)
	_ = d.Set("name", snapshot.Name)
	_ = d.Set("resource_uri", snapshot.ResourceURI)
	_ = d.Set("status", snapshot.Status)
	_ = d.Set("timestamp", snapshot.Timestamp)

	return nil
}

func resourceCloudSigmaSnapshotUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	snapshot := &cloudsigma.Snapshot{
		Drive: cloudsigma.Drive{},
		UUID:  d.Id(),
	}

	if drive, ok := d.GetOk("drive"); ok {
		snapshot.Drive.UUID = drive.(string)
	}
	if name, ok := d.GetOk("name"); ok {
		snapshot.Name = name.(string)
	}

	updateRequest := &cloudsigma.SnapshotUpdateRequest{
		Snapshot: snapshot,
	}

	_, _, err := client.Snapshots.Update(context.Background(), snapshot.UUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update snapshot: %s", err)
	}

	return resourceCloudSigmaSnapshotRead(d, meta)
}

func resourceCloudSigmaSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	_, err := client.Snapshots.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting snapshot: %s", err)
	}

	d.SetId("")

	return nil
}
