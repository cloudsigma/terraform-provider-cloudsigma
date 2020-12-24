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
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(250, 124000)), // 250MHz - 100GHz
			},

			"drive": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"ipv4_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"memory": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(268435456, 137438953472)), // 256MB - 128GB
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"ssh_keys": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.NoZeroValues),
				},
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

	if v, ok := d.GetOk("ssh_keys"); ok {
		sshKeys := expandSSHKeys(v.(*schema.Set).List())
		createRequest.Servers[0].PublicKeys = sshKeys
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

	// attach drives
	if ds, ok := d.GetOk("drive"); ok {
		serverDrives := server.Drives

		drives := ds.([]interface{})
		for _, dr := range drives {
			drive := dr.(map[string]interface{})

			serverDrives = append(serverDrives, cloudsigma.ServerDrive{
				BootOrder:  len(serverDrives),
				DevChannel: fmt.Sprintf("0:%d", len(serverDrives)),
				Device:     "virtio",
				Drive:      &cloudsigma.Drive{UUID: drive["uuid"].(string)},
			})
		}

		attachRequest := &cloudsigma.ServerAttachDriveRequest{
			CPU:         server.CPU,
			Drives:      serverDrives,
			Memory:      server.Memory,
			Name:        server.Name,
			VNCPassword: server.VNCPassword,
		}
		log.Printf("[DEBUG] Server attach drive configuration: %#v", *attachRequest)
		_, _, err := client.Servers.AttachDrive(ctx, d.Id(), attachRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

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
	_ = d.Set("ipv4_address", findIPv4Address(server, "public"))
	_ = d.Set("memory", server.Memory)
	_ = d.Set("name", server.Name)
	_ = d.Set("resource_uri", server.ResourceURI)
	_ = d.Set("vnc_password", server.VNCPassword)

	return nil
}

func resourceCloudSigmaServerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	updateRequest := &cloudsigma.ServerUpdateRequest{
		Server: &cloudsigma.Server{
			CPU:         d.Get("cpu").(int),
			Memory:      d.Get("memory").(int),
			Name:        d.Get("name").(string),
			VNCPassword: d.Get("vnc_password").(string),
		},
	}

	if d.HasChange("drive") {
		serverDrives := make([]cloudsigma.ServerDrive, 0)

		drives := d.Get("drive").([]interface{})
		for _, dr := range drives {
			drive := dr.(map[string]interface{})

			serverDrives = append(serverDrives, cloudsigma.ServerDrive{
				BootOrder:  len(serverDrives),
				DevChannel: fmt.Sprintf("0:%d", len(serverDrives)),
				Device:     "virtio",
				Drive:      &cloudsigma.Drive{UUID: drive["uuid"].(string)},
			})
		}

		updateRequest.Drives = serverDrives
	}

	err := stopServer(ctx, client, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Server update configuration: %#v", *updateRequest)
	_, _, err = client.Servers.Update(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	err = startServer(ctx, client, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceCloudSigmaServerRead(ctx, d, meta)
}

func resourceCloudSigmaServerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	server, resp, err := client.Servers.Get(ctx, d.Id())
	if err != nil {
		// handle remotely destroyed server
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// Stop server
	err = stopServer(ctx, client, server.UUID)
	if err != nil {
		return diag.Errorf("error stopping server: %s", err)
	}

	// Delete server
	_, err = client.Servers.Delete(ctx, server.UUID)
	if err != nil {
		return diag.Errorf("error deleting server: %s", err)
	}

	d.SetId("")

	return nil
}

func expandSSHKeys(sshKeys []interface{}) []cloudsigma.Keypair {
	expandedSshKeys := make([]cloudsigma.Keypair, len(sshKeys))
	for i, s := range sshKeys {
		sshKey := s.(string)
		var expandedSshKey cloudsigma.Keypair
		expandedSshKey.UUID = sshKey
		expandedSshKeys[i] = expandedSshKey
	}

	return expandedSshKeys
}

func findIPv4Address(server *cloudsigma.Server, addrType string) string {
	if server.Runtime == nil {
		return ""
	}

	for _, nic := range server.Runtime.RuntimeNICs {
		if nic.InterfaceType == addrType {
			return nic.IPv4.UUID
		}
	}
	return ""
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
	server, resp, err := client.Servers.Get(ctx, serverUUID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}
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
