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

func resourceCloudSigmaServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudSigmaServerCreate,
		ReadContext:   resourceCloudSigmaServerRead,
		UpdateContext: resourceCloudSigmaServerUpdate,
		DeleteContext: resourceCloudSigmaServerDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"cpu": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(250, 124000), // 256MB - 128GB
			},

			"memory": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(268435456, 137438953472), // 256MB - 128GB
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vnc_password": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceCloudSigmaServerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

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
		return diag.FromErr(err)
	}

	server := servers[0]
	log.Printf("[INFO] Server ID: %s", server.UUID)

	// store the resulting UUID so we can look this up later
	d.SetId(server.UUID)

	// start server
	err = startServer(ctx, client, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaServerRead(ctx, d, meta)
}

func resourceCloudSigmaServerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	server, resp, err := client.Servers.Get(ctx, d.Id())
	if err != nil {
		// If the server is somehow already destroyed, mark as successfully gone
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("cpu", server.CPU)
	_ = d.Set("memory", server.Memory)
	_ = d.Set("name", server.Name)
	_ = d.Set("resource_uri", server.ResourceURI)
	_ = d.Set("vnc_password", server.VNCPassword)

	return nil
}

func resourceCloudSigmaServerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

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
			return diag.FromErr(err)
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
		err := stopServer(ctx, client, d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		// update with new values
		_, _, err = client.Servers.Update(ctx, d.Id(), updateRequest)
		if err != nil {
			return diag.FromErr(err)
		}
		// start server again
		err = startServer(ctx, client, d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceCloudSigmaServerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	// Stop server
	err := stopServer(ctx, client, d.Id())
	if err != nil {
		return diag.Errorf("error stopping server: %s", err)
	}

	// Delete server
	_, err = client.Servers.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting server: %s", err)
	}

	d.SetId("")

	return nil
}

func serverStateRefreshFunc(ctx context.Context, client *cloudsigma.Client, serverUUID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		server, _, err := client.Servers.Get(ctx, serverUUID)
		if err != nil {
			return nil, "", fmt.Errorf("error retrieving server with uuid %s: %s", serverUUID, err)
		}

		return server, server.Status, nil
	}
}

func startServer(ctx context.Context, client *cloudsigma.Client, serverUUID string) error {
	log.Printf("[DEBUG] Starting server (%s)", serverUUID)

	log.Printf("[DEBUG] Checking server status before starting")
	server, _, err := client.Servers.Get(ctx, serverUUID)
	if err != nil {
		return fmt.Errorf("error retrieving server: %s", err)
	}

	if server.Status == "running" {
		log.Printf("[DEBUG] Server (%s) is already running", server.UUID)
		return nil
	}

	_, _, err = client.Servers.Start(ctx, server.UUID)
	if err != nil {
		return fmt.Errorf("error starting server: %s", err)
	}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"stopped", "starting"},
		Target:     []string{"running"},
		Refresh:    serverStateRefreshFunc(ctx, client, server.UUID),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("error waiting for server (%s) to become running: %s", server.UUID, err)
	}

	return nil
}

func stopServer(ctx context.Context, client *cloudsigma.Client, serverUUID string) error {
	log.Printf("[DEBUG] Stopping server (%s)", serverUUID)

	log.Printf("[DEBUG] Checking server status before stopping")
	server, _, err := client.Servers.Get(ctx, serverUUID)
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
		Refresh:    serverStateRefreshFunc(ctx, client, server.UUID),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(context.Background()); err != nil {
		return fmt.Errorf("error waiting for server (%s) to become stopped: %s", server.UUID, err)
	}

	return nil
}
