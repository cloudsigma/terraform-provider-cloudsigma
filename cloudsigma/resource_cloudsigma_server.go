package cloudsigma

import (
	"context"
	"fmt"
	"log"
	"regexp"
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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
		},

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

			"enclave_page_caches": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
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

			"meta": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringDoesNotMatch(regexp.MustCompile("^ssh_public_key$"),
						"Do not specify ssh_public_key in the meta. Use ssh_keys property instead.")),
				},
				ValidateDiagFunc: validation.MapKeyLenBetween(0, 32),
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"network": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv4_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"dhcp", "static"}, false)),
						},
						"vlan_uuid": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"smp": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"ssh_keys": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.NoZeroValues,
				},
			},

			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.NoZeroValues,
				},
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

	err := validateSMP(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// build create configuration
	createRequest := &cloudsigma.ServerCreateRequest{
		Servers: []cloudsigma.Server{
			{
				CPU:         d.Get("cpu").(int),
				Memory:      d.Get("memory").(int),
				Name:        d.Get("name").(string),
				SMP:         d.Get("smp").(int),
				VNCPassword: d.Get("vnc_password").(string),
			},
		},
	}

	if v, ok := d.GetOk("enclave_page_caches"); ok {
		createRequest.Servers[0].EnclavePageCaches = expandEnclavePageCaches(v.([]interface{}))
	}

	if ns, ok := d.GetOk("network"); ok {
		networks := ns.([]interface{})
		createRequest.Servers[0].NICs = make([]cloudsigma.ServerNIC, len(networks))

		for i, n := range networks {
			network := n.(map[string]interface{})
			networkType := network["type"].(string)
			networkAddress := network["ipv4_address"].(string)
			networkVlan := network["vlan_uuid"].(string)

			if networkType == "static" && networkAddress == "" {
				return diag.Errorf("network address cannot be empty if type is static")
			}
			if networkType != "" && networkVlan != "" {
				return diag.Errorf("cannot assign both network type and vlan")
			}

			if networkType == "static" {
				conf := &cloudsigma.ServerIPConfiguration{
					Type:      networkType,
					IPAddress: &cloudsigma.IP{UUID: networkAddress},
				}
				createRequest.Servers[0].NICs[i].IP4Configuration = conf
			} else if networkType == "dhcp" {
				conf := &cloudsigma.ServerIPConfiguration{
					Type: networkType,
				}
				createRequest.Servers[0].NICs[i].IP4Configuration = conf
			} else if networkVlan != "" {
				vlan := &cloudsigma.VLAN{
					UUID: networkVlan,
				}
				createRequest.Servers[0].NICs[i].VLAN = vlan
			}
		}
	} else {
		createRequest.Servers[0].NICs = []cloudsigma.ServerNIC{
			{
				IP4Configuration: &cloudsigma.ServerIPConfiguration{Type: "dhcp"},
				Model:            "virtio",
			},
		}
	}

	if v, ok := d.GetOk("ssh_keys"); ok {
		createRequest.Servers[0].PublicKeys = expandSSHKeys(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("tags"); ok {
		createRequest.Servers[0].Tags = expandTags(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("meta"); ok {
		m := v.(map[string]interface{})
		if createRequest.Servers[0].Meta == nil {
			createRequest.Servers[0].Meta = make(map[string]interface{})
		}

		for k, val := range m {
			createRequest.Servers[0].Meta[k] = val
		}
	}

	log.Printf("[DEBUG] Server create configuration: %v", createRequest)
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

		server.Drives = serverDrives
		updateRequest := &cloudsigma.ServerUpdateRequest{
			Server: &server,
		}
		log.Printf("[DEBUG] Server update configuration (attach drives): %v", updateRequest)
		_, _, err := client.Servers.Update(ctx, d.Id(), updateRequest)
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
	_ = d.Set("smp", server.SMP)
	_ = d.Set("vnc_password", server.VNCPassword)

	if server.PublicKeys != nil {
		_ = d.Set("ssh_keys", extractSSHKeys(server.PublicKeys))
	}

	if server.Meta != nil {
		meta := make(map[string]interface{})
		for k, val := range server.Meta {
			// Ignore ssh_public key as it is managed by ssh_keys property
			if k != "ssh_public_key" {
				meta[k] = val.(string)
			}
		}
		if len(meta) > 0 {
			_ = d.Set("meta", meta)
		}
	}

	if len(server.NICs) > 0 {
		var networks []map[string]interface{}
		for _, nws := range server.NICs {
			nw := make(map[string]interface{})
			if nws.IP4Configuration != nil {
				nw["type"] = nws.IP4Configuration.Type
				if nws.IP4Configuration.IPAddress != nil {
					nw["ipv4_address"] = nws.IP4Configuration.IPAddress.UUID
				}
			}
			if nws.VLAN != nil {
				nw["vlan_uuid"] = nws.VLAN.UUID
			}
			networks = append(networks, nw)
		}
		if err := d.Set("network", networks); err != nil {
			return diag.Errorf("error setting network: %v", err)
		}
	}

	if err := d.Set("enclave_page_caches", flattenEnclavePageCaches(server.EnclavePageCaches)); err != nil {
		return diag.Errorf("error setting Server EPC - error: %#v", err)
	}

	if err := d.Set("tags", flattenTags(server.Tags)); err != nil {
		return diag.Errorf("error setting Server tags - error: %#v", err)
	}

	return nil
}

func resourceCloudSigmaServerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	// Note that if a server is running, only name, meta, and tags fields can be changed
	// and all other changes to the definition of a running server will be ignored.
	needRestart := d.HasChangesExcept("name", "meta", "tags")

	err := validateSMP(d)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := &cloudsigma.ServerUpdateRequest{
		Server: &cloudsigma.Server{
			CPU:         d.Get("cpu").(int),
			Memory:      d.Get("memory").(int),
			Name:        d.Get("name").(string),
			SMP:         d.Get("smp").(int),
			VNCPassword: d.Get("vnc_password").(string),
		},
	}

	if v, ok := d.GetOk("enclave_page_caches"); ok {
		updateRequest.EnclavePageCaches = expandEnclavePageCaches(v.([]interface{}))
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

	if v, ok := d.GetOk("meta"); ok {
		m := v.(map[string]interface{})
		if updateRequest.Meta == nil {
			updateRequest.Meta = make(map[string]interface{})
		}

		for k, val := range m {
			updateRequest.Meta[k] = val
		}
	}

	if d.HasChange("network") {
		serverNICs := make([]cloudsigma.ServerNIC, 0)

		networks := d.Get("network").([]interface{})
		for _, n := range networks {
			network := n.(map[string]interface{})
			networkType := network["type"].(string)
			networkAddress := network["ipv4_address"].(string)
			networkVlan := network["vlan_uuid"].(string)

			if networkType == "static" && networkAddress == "" {
				return diag.Errorf("network address cannot be empty if type is static")
			}
			if networkType != "" && networkVlan != "" {
				return diag.Errorf("cannot assign both network type and vlan")
			}

			if networkType == "static" {
				serverNICs = append(serverNICs, cloudsigma.ServerNIC{
					IP4Configuration: &cloudsigma.ServerIPConfiguration{
						Type:      networkType,
						IPAddress: &cloudsigma.IP{UUID: networkAddress},
					},
				})
			} else if networkType == "dhcp" {
				serverNICs = append(serverNICs, cloudsigma.ServerNIC{
					IP4Configuration: &cloudsigma.ServerIPConfiguration{
						Type: networkType,
					},
				})
			} else if networkVlan != "" {
				serverNICs = append(serverNICs, cloudsigma.ServerNIC{
					VLAN: &cloudsigma.VLAN{
						UUID: networkVlan,
					},
				})
			}
		}

		updateRequest.NICs = serverNICs
	}

	if v, ok := d.GetOk("tags"); ok {
		updateRequest.Tags = expandTags(v.(*schema.Set).List())
	}

	if needRestart {
		err = stopServer(ctx, client, d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] Server update configuration: %v", *updateRequest)
	_, _, err = client.Servers.Update(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	if needRestart {
		err = startServer(ctx, client, d.Id())
		if err != nil {
			return diag.FromErr(err)
		}
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

func extractSSHKeys(serverSSHKeys []cloudsigma.Keypair) []interface{} {
	extractedSshKeys := make([]interface{}, len(serverSSHKeys))
	for i, v := range serverSSHKeys {
		extractedSshKeys[i] = v.UUID
	}

	return extractedSshKeys
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

func validateSMP(d *schema.ResourceData) error {
	if v, ok := d.GetOk("smp"); ok {
		smp := v.(int)
		cpu := d.Get("cpu").(int)
		if cpu/smp < 1000 {
			return fmt.Errorf("the minimum amount of cpu per smp is 1000 (currently is %v)", cpu/smp)
		}
	}
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
