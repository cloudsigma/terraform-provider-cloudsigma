package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudSigmaServerCreate,
		Read:   resourceCloudSigmaServerRead,
		Update: resourceCloudSigmaServerUpdate,
		Delete: resourceCloudSigmaServerDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"cpu": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"memory": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"vnc_password": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceCloudSigmaServerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)
	ctx := context.Background()

	log.Printf("[DEBUG] Creating CloudSigma server...")
	createRequest := &cloudsigma.ServerCreateRequest{
		Servers: []cloudsigma.Server{
			{
				CPU:         d.Get("cpu").(int),
				Memory:      d.Get("memory").(int) * 1024 * 1024,
				Name:        d.Get("name").(string),
				VNCPassword: d.Get("vnc_password").(string),
			},
		},
	}
	servers, _, err := client.Servers.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error creating server: %s", err)
	}

	// we create only one server
	server := servers[0]
	log.Printf("[INFO] Server ID: %s", server.UUID)
	d.SetId(server.UUID)

	log.Printf("[DEBUG] Starting CloudSigma server (%s)...", server.UUID)
	// Wait for the server to become starting
	_, _, err = client.Servers.Start(ctx, server.UUID)
	if err != nil {
		return fmt.Errorf("error starting server: %s", err)
	}
	stateConf := &resource.StateChangeConf{
		Pending: []string{"stopped", "starting"},
		Target:  []string{"running"},
		Refresh: serverStateRefreshFunc(client, server.UUID),
		Timeout: 10 * time.Minute,
		Delay:   10 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("error waiting for server (%s) to become running: %s", server.UUID, err)
	}

	return resourceCloudSigmaServerRead(d, meta)
}

func resourceCloudSigmaServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	server, resp, err := client.Servers.Get(context.Background(), d.Id())
	if err != nil {
		// If the server is somehow already destroyed, mark as successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving server: %s", err)
	}

	_ = d.Set("cpu", server.CPU)
	_ = d.Set("memory", server.Memory/1024/1024)
	_ = d.Set("name", server.Name)
	_ = d.Set("vnc_password", server.VNCPassword)

	return nil
}

func resourceCloudSigmaServerUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceCloudSigmaSSHKeyRead(d, meta)
}

func resourceCloudSigmaServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)
	ctx := context.Background()

	log.Printf("[INFO] Stopping server: %s", d.Id())
	_, _, err := client.Servers.Stop(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("error stopping server: %s", err)
	}
	// Wait for the server to become 'stopped'
	stateConf := &resource.StateChangeConf{
		Pending: []string{"running", "stopping"},
		Target:  []string{"stopped"},
		Refresh: serverStateRefreshFunc(client, d.Id()),
		Timeout: 10 * time.Minute,
		Delay:   10 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("error waiting for server (%s) to become stopped: %s", d.Id(), err)
	}

	_, err = client.Servers.Delete(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("error deleting server: %s", err)
	}

	d.SetId("")

	return nil
}

func serverStateRefreshFunc(client *cloudsigma.Client, serverUUID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		server, _, err := client.Servers.Get(context.Background(), serverUUID)
		if err != nil {
			return nil, "", fmt.Errorf("error retrieving server with uuid %s: %s", serverUUID, err)
		}

		return server, server.Status, nil
	}
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
