package cloudsigma

import (
	"context"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudSigmaRemoteSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: `
The remote snapshot resource allows you to manage CloudSigma remote snapshots.

Remote snapshots are point-in-time versions of a drive. They can be cloned to
a full drive, which makes it possible to restore an older version of a VM image.
`,

		CreateContext: resourceCloudSigmaRemoteSnapshotCreate,
		ReadContext:   resourceCloudSigmaRemoteSnapshotRead,
		UpdateContext: resourceCloudSigmaRemoteSnapshotUpdate,
		DeleteContext: resourceCloudSigmaRemoteSnapshotDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"drive": {
				Description: "The UUID of the drive.",
				Type:        schema.TypeString,
				Required:    true,
			},

			"location": {
				Description: "The location of the remote snapshot.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"name": {
				Description: "The remote snapshot name.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"resource_uri": {
				Description: "The unique resource identifier of the remote snapshot.",
				Computed:    true,
				Type:        schema.TypeString,
			},

			"status": {
				Description: "The remote snapshot status.",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceCloudSigmaRemoteSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	createRequest := &cloudsigma.RemoteSnapshotCreateRequest{
		// Drive: &cloudsigma.Drive{
		// 	UUID: d.Get("drive").(string),
		// },
		// Location: d.Get("location").(string),
		// Name:     d.Get("name").(string),
	}
	log.Printf("[DEBUG] Remote snapshot create configuration: %#v", *createRequest)
	remoteSnapshots, _, err := client.RemoteSnapshots.Create(ctx, createRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(remoteSnapshots[0].UUID)
	log.Printf("[INFO] Remote snapshot ID: %s", d.Id())

	return resourceCloudSigmaRemoteSnapshotRead(ctx, d, meta)
}

func resourceCloudSigmaRemoteSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	remoteSnapshot, resp, err := client.RemoteSnapshots.Get(ctx, d.Id())
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("drive", remoteSnapshot.Drive.UUID)
	_ = d.Set("name", remoteSnapshot.Name)
	_ = d.Set("resource_uri", remoteSnapshot.ResourceURI)
	_ = d.Set("status", remoteSnapshot.Status)

	return nil
}

func resourceCloudSigmaRemoteSnapshotUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	remoteSnapshot := &cloudsigma.RemoteSnapshot{
		Snapshot: cloudsigma.Snapshot{
			UUID: d.Id(),
		},
	}

	if d.HasChange("name") {
		_, newName := d.GetChange("name")
		remoteSnapshot.Snapshot.Name = newName.(string)
	}

	updateRequest := &cloudsigma.RemoteSnapshotUpdateRequest{
		RemoteSnapshot: remoteSnapshot,
	}
	log.Printf("[DEBUG] Remote snapshot update configuration: %#v", *updateRequest)
	_, _, err := client.RemoteSnapshots.Update(ctx, remoteSnapshot.UUID, updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaRemoteSnapshotRead(ctx, d, meta)
}

func resourceCloudSigmaRemoteSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	_, err := client.RemoteSnapshots.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
