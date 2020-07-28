package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceCloudSigmaDriveAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudSigmaDriveAttachmentCreate,
		ReadContext:   resourceCloudSigmaDriveAttachmentRead,
		DeleteContext: resourceCloudSigmaDriveAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"drive_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"server_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func resourceCloudSigmaDriveAttachmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	driveUUID := d.Get("drive_id").(string)
	_, _, err := client.Drives.Get(ctx, driveUUID)
	if err != nil {
		return diag.Errorf("error getting drive: %s", err)
	}

	serverUUID := d.Get("server_id").(string)
	server, _, err := client.Servers.Get(ctx, serverUUID)
	if err != nil {
		return diag.Errorf("error getting server: %s", err)
	}
	if server.Status != "stopped" {
		_, _, err := client.Servers.Stop(ctx, server.UUID)
		if err != nil {
			return diag.Errorf("error stopping server: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending: []string{"running", "starting", "stopping"},
			Target:  []string{"stopped"},
			Refresh: serverStateRefreshFunc(ctx, client, server.UUID),
			Timeout: 10 * time.Minute,
			Delay:   5 * time.Second,
		}
		if _, err := stateConf.WaitForStateContext(ctx); err != nil {
			return diag.Errorf("error waiting for server (%s) to become running: %s", server.UUID, err)
		}
	}

	serverDrives := server.Drives
	serverDrives = append(serverDrives, cloudsigma.ServerDrive{
		BootOrder:  len(serverDrives),
		DevChannel: fmt.Sprintf("0:%d", len(serverDrives)),
		Device:     "virtio",
		Drive:      &cloudsigma.Drive{UUID: driveUUID},
	})

	attachRequest := &cloudsigma.ServerAttachDriveRequest{
		CPU:         server.CPU,
		Drives:      serverDrives,
		Memory:      server.Memory,
		Name:        server.Name,
		VNCPassword: server.VNCPassword,
	}

	log.Printf("[DEBUG] Attaching Drive (%s) to Server (%s)", driveUUID, serverUUID)
	_, _, err = client.Servers.AttachDrive(ctx, server.UUID, attachRequest)
	if err != nil {
		return diag.Errorf("error attaching drive: %s", err)
	}

	d.SetId(driveAttachmentID(driveUUID, serverUUID))

	return resourceCloudSigmaDriveAttachmentRead(ctx, d, meta)
}

func resourceCloudSigmaDriveAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceCloudSigmaDriveAttachmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	driveUUID := d.Get("drive_id").(string)

	serverUUID := d.Get("server_id").(string)
	server, _, err := client.Servers.Get(ctx, serverUUID)
	if err != nil {
		return diag.Errorf("error getting server: %s", err)
	}

	// exclude our drive
	serverDrives := make([]cloudsigma.ServerDrive, 0)
	for _, serverDrive := range server.Drives {
		if serverDrive.Drive.UUID != driveUUID {
			serverDrives = append(serverDrives, serverDrive)
		}
	}

	attachRequest := &cloudsigma.ServerAttachDriveRequest{
		CPU:         server.CPU,
		Drives:      serverDrives,
		Memory:      server.Memory,
		Name:        server.Name,
		VNCPassword: server.VNCPassword,
	}

	log.Printf("[DEBUG] Attaching Drive (%s) to Server (%s)", driveUUID, serverUUID)
	_, _, err = client.Servers.AttachDrive(ctx, server.UUID, attachRequest)
	if err != nil {
		return diag.Errorf("error attaching drive: %s", err)
	}

	d.SetId("")

	return nil
}

func driveAttachmentID(driveUUID, serverUUID string) string {
	return fmt.Sprintf("serverdrive-%s-%s", serverUUID, driveUUID)
}
