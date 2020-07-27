package cloudsigma

import (
	"context"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaSnapshot() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudSigmaSnapshotCreate,
		ReadContext:   resourceCloudSigmaSnapshotRead,
		UpdateContext: resourceCloudSigmaSnapshotUpdate,
		DeleteContext: resourceCloudSigmaSnapshotDelete,

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

func resourceCloudSigmaSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	// check if drive status is mounted or unmounted
	driveUUID := d.Get("drive").(string)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"cloning_dst", "creating"},
		Target:     []string{"mounted", "unmounted"},
		Refresh:    driveStateRefreshFunc(ctx, client, driveUUID),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return diag.FromErr(err)
	}

	createRequest := &cloudsigma.SnapshotCreateRequest{
		Snapshots: []cloudsigma.Snapshot{
			{
				Drive: &cloudsigma.Drive{
					UUID: d.Get("drive").(string),
				},
				Name: d.Get("name").(string),
			},
		},
	}
	log.Printf("[DEBUG] Snapshot create configuration: %#v", *createRequest)
	snapshots, _, err := client.Snapshots.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(snapshots[0].UUID)
	log.Printf("[INFO] Snapshot ID: %s", d.Id())

	return resourceCloudSigmaSnapshotRead(ctx, d, meta)
}

func resourceCloudSigmaSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	snapshot, resp, err := client.Snapshots.Get(ctx, d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("drive", snapshot.Drive.UUID)
	_ = d.Set("name", snapshot.Name)
	_ = d.Set("resource_uri", snapshot.ResourceURI)
	_ = d.Set("status", snapshot.Status)
	_ = d.Set("timestamp", snapshot.Timestamp)

	return nil
}

func resourceCloudSigmaSnapshotUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	snapshot := &cloudsigma.Snapshot{
		Drive: &cloudsigma.Drive{},
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
	log.Printf("[DEBUG] Snapshot update configuration: %#v", *updateRequest)
	_, _, err := client.Snapshots.Update(ctx, snapshot.UUID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaSnapshotRead(ctx, d, meta)
}

func resourceCloudSigmaSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	_, err := client.Snapshots.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
