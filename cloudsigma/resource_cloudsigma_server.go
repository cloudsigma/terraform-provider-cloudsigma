package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

	// build create configuration
	createRequest := &cloudsigma.ServerCreateRequest{
		Servers: []cloudsigma.Server{
			{
				CPU:    d.Get("cpu").(int),
				Memory: d.Get("memory").(int),
				Name:   d.Get("name").(string),
				NICs: []cloudsigma.ServerNIC{
					{
						IP4Configuration: &cloudsigma.ServerIPConfiguration{Type: "dhcp"},
						Model:            "virtio",
					},
				},
				VNCPassword: d.Get("vnc_password").(string),
			},
		},
	}
	log.Printf("[DEBUG] Server create configuration: %#v", *createRequest)
	servers, _, err := client.Servers.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error creating server: %s", err)
	}

	server := servers[0]
	log.Printf("[INFO] Server ID: %s", server.UUID)

	// store the resulting UUID so we can look this up later
	d.SetId(server.UUID)

	// start server
	err = startServer(client, d.Id())
	if err != nil {
		return fmt.Errorf("error starting server: %s", err)
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
	_ = d.Set("memory", server.Memory)
	_ = d.Set("name", server.Name)
	_ = d.Set("vnc_password", server.VNCPassword)

	return nil
}

func resourceCloudSigmaServerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)
	ctx := context.Background()

	if d.HasChange("name") {
		// we don't need to stop server when updating 'name'
		_, newName := d.GetChange("name")
		updateRequest := &cloudsigma.ServerUpdateRequest{
			Server: &cloudsigma.Server{
				CPU:         d.Get("cpu").(int),
				Memory:      d.Get("memory").(int),
				Name:        newName.(string),
				VNCPassword: d.Get("vnc_password").(string),
			},
		}
		log.Printf("[DEBUG] Server update configuration: %#v", *updateRequest)
		_, _, err := client.Servers.Update(ctx, d.Id(), updateRequest)
		if err != nil {
			return fmt.Errorf("error updating server: %s", err)
		}
	}

	if d.HasChanges("cpu", "memory", "vnc_password") {
		// we need to stop server when updating 'cpu', 'memory' or 'vnc_password'
		_, newCPU := d.GetChange("cpu")
		_, newMemory := d.GetChange("memory")
		_, newVNCPassword := d.GetChange("vnc_password")
		updateRequest := &cloudsigma.ServerUpdateRequest{
			Server: &cloudsigma.Server{
				CPU:         newCPU.(int),
				Memory:      newMemory.(int),
				Name:        d.Get("name").(string),
				VNCPassword: newVNCPassword.(string),
			},
		}
		log.Printf("[DEBUG] Server update configuration: %#v", *updateRequest)
		// stop server first
		err := stopServer(client, d.Id())
		if err != nil {
			return fmt.Errorf("error stopping server: %s", err)
		}
		// update with new values
		_, _, err = client.Servers.Update(ctx, d.Id(), updateRequest)
		if err != nil {
			return fmt.Errorf("error updating server: %s", err)
		}
		// start server again
		err = startServer(client, d.Id())
		if err != nil {
			return fmt.Errorf("error starting server: %s", err)
		}
	}

	return nil
}

func resourceCloudSigmaServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)
	ctx := context.Background()

	// Stop server
	err := stopServer(client, d.Id())
	if err != nil {
		return fmt.Errorf("error stopping server: %s", err)
	}

	// Delete server
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

func startServer(client *cloudsigma.Client, serverUUID string) error {
	log.Printf("[DEBUG] Starting server (%s)", serverUUID)

	log.Printf("[DEBUG] Checking server status before starting")
	server, _, err := client.Servers.Get(context.Background(), serverUUID)
	if err != nil {
		return fmt.Errorf("error retrieving server: %s", err)
	}

	if server.Status == "running" {
		log.Printf("[DEBUG] Server (%s) is already running", server.UUID)
		return nil
	}

	_, _, err = client.Servers.Start(context.Background(), server.UUID)
	if err != nil {
		return fmt.Errorf("error starting server: %s", err)
	}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"stopped", "starting"},
		Target:     []string{"running"},
		Refresh:    serverStateRefreshFunc(client, server.UUID),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(context.Background()); err != nil {
		return fmt.Errorf("error waiting for server (%s) to become running: %s", server.UUID, err)
	}

	return nil
}

func stopServer(client *cloudsigma.Client, serverUUID string) error {
	log.Printf("[DEBUG] Stopping server (%s)", serverUUID)

	log.Printf("[DEBUG] Checking server status before stopping")
	server, _, err := client.Servers.Get(context.Background(), serverUUID)
	if err != nil {
		return fmt.Errorf("error retrieving server: %s", err)
	}

	if server.Status == "stopped" {
		log.Printf("[DEBUG] Server (%s) is already stopped", server.UUID)
		return nil
	}

	_, _, err = client.Servers.Stop(context.Background(), server.UUID)
	if err != nil {
		return fmt.Errorf("error stopping server: %s", err)
	}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"running", "stopping"},
		Target:     []string{"stopped"},
		Refresh:    serverStateRefreshFunc(client, server.UUID),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(context.Background()); err != nil {
		return fmt.Errorf("error waiting for server (%s) to become stopped: %s", server.UUID, err)
	}

	return nil
}
