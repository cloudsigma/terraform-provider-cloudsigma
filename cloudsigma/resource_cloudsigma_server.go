package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCloudSigmaServerKey() *schema.Resource {
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

			"drive": {
				Type:     schema.TypeString,
				Optional: true,
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

	log.Printf("[DEBUG] Cloning library drive...")
	drive, err := cloneDrive(d, client)
	if err != nil {
		return fmt.Errorf("error creating library drive: %s", err)
	}

	log.Printf("[DEBUG] Creating CloudSigma server...")
	server, err := createServer(d, drive, client)
	if err != nil {
		return fmt.Errorf("error creating server: %s", err)
	}

	log.Printf("[DEBUG] Starting CloudSigma server...")
	err = startServer(server.UUID, client)
	if err != nil {
		return err
	}

	d.SetId(server.UUID)
	log.Printf("[INFO] Server: %s", server.UUID)
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
	// _ = d.Set("drive", server.Drives[0].DriveUUID)
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

	log.Printf("[INFO] Deleting server: %s", d.Id())
	err := stopServer(d.Id(), client)
	if err != nil {
		return fmt.Errorf("error stopping server: %s", err)
	}
	_, err = client.Servers.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting server: %s", err)
	}

	d.SetId("")

	return nil
}

func cloneDrive(d *schema.ResourceData, client *cloudsigma.Client) (*cloudsigma.Drive, error) {
	uuid := d.Get("drive").(string)
	driveCloneRequest := &cloudsigma.DriveCloneRequest{
		Name:        d.Get("name").(string),
		Size:        10 * 1024 * 1024 * 1024,
		StorageType: "dssd",
	}

	log.Printf("[DEBUG] SSH key create configuration: %#v", driveCloneRequest)
	clonedDrive, _, err := client.LibraryDrives.Clone(context.Background(), uuid, driveCloneRequest)
	if err != nil {
		return clonedDrive, err
	}

	log.Printf("[DEBUG] Waiting until cloning process are done...")
	driveUUID := clonedDrive.UUID
	for {
		drive, _, err := client.Drives.Get(context.Background(), driveUUID)
		if err != nil {
			return clonedDrive, nil
		}

		if drive.Status == "unmounted" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return clonedDrive, nil
}

func createServer(d *schema.ResourceData, drive *cloudsigma.Drive, client *cloudsigma.Client) (*cloudsigma.Server, error) {
	serverCreateRequest := &cloudsigma.ServerCreateRequest{
		CPU:         d.Get("cpu").(int),
		CPUType:     "amd",
		Memory:      d.Get("memory").(int) * 1024 * 1024,
		Name:        d.Get("name").(string),
		VNCPassword: d.Get("vnc_password").(string),
	}

	log.Printf("[DEBUG] Server create configuration: %#v", serverCreateRequest)
	server, _, err := client.Servers.Create(context.Background(), serverCreateRequest)
	if err != nil {
		return server, err
	}

	attachDriveRequest := &cloudsigma.AttachDriveRequest{
		CPU:     server.CPU,
		CPUType: server.CPUType,
		Drives: []cloudsigma.ServerDrive{
			{BootOrder: 1, DevChannel: "0:0", Device: "virtio", DriveUUID: drive.UUID},
		},
		Memory:      server.Memory,
		Name:        server.Name,
		VNCPassword: server.VNCPassword,
	}

	log.Printf("[DEBUG] Attaching existing drive to virtual server...")
	server, _, err = client.Servers.AttachDrive(context.Background(), server.UUID, attachDriveRequest)
	if err != nil {
		return server, err
	}

	return server, nil
}

func startServer(serverUUID string, client *cloudsigma.Client) error {
	log.Printf("[DEBUG] Checking server state...")
	server, _, err := client.Servers.Get(context.Background(), serverUUID)
	if err != nil {
		return nil
	}
	if server.Status == "running" {
		log.Printf("[DEBUG] Server is already running")
		return nil
	}

	log.Printf("[INFO] Starting CloudSigma virtual server...")
	_, _, err = client.Servers.Start(context.Background(), serverUUID)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting until server is running...")
	for {
		server, _, err = client.Servers.Get(context.Background(), serverUUID)
		if err != nil {
			return nil
		}
		if server.Status == "running" {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func stopServer(serverUUID string, client *cloudsigma.Client) error {
	log.Printf("[DEBUG] Checking server state...")
	server, _, err := client.Servers.Get(context.Background(), serverUUID)
	if err != nil {
		return nil
	}
	if server.Status == "stopped" {
		log.Printf("[DEBUG] Server is already stopped")
		return nil
	}

	log.Printf("[DEBUG] Stopping CloudSigma virtual server...")
	_, _, err = client.Servers.Shutdown(context.Background(), serverUUID)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting until server is stopped...")
	for {
		server, _, err := client.Servers.Get(context.Background(), serverUUID)
		if err != nil {
			return err
		}
		if server.Status == "running" {
			return fmt.Errorf("could not stop server %v", serverUUID)
		}
		if server.Status == "stopped" {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}
