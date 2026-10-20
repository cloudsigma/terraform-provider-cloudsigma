package cloudsigma

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

func readNativeServer(ctx context.Context, d *schema.ResourceData, client *cloudsigmaapi.Client) diag.Diagnostics {
	if e := ctx.Err(); e != nil {
		return diag.FromErr(e)
	}
	server, err := client.Server(ctx, d.Id())
	if e := ctx.Err(); e != nil {
		return diag.FromErr(e)
	}
	if err != nil {
		var apiError *cloudsigmaapi.Error
		if errors.As(err, &apiError) && apiError.Status == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	metadata, metadataErr := serverReadMetadata(server.Meta)
	if metadataErr != nil {
		return diag.FromErr(metadataErr)
	}

	_ = d.Set("cpu", server.CPU)
	_ = d.Set("ipv4_address", nativeServerIPv4(server))
	_ = d.Set("memory", server.Memory)
	_ = d.Set("name", server.Name)
	_ = d.Set("resource_uri", server.ResourceURI)
	_ = d.Set("smp", server.SMP)
	_ = d.Set("vnc_password", server.VNCPassword)

	if server.PublicKeys != nil {
		_ = d.Set("ssh_keys", nativeServerKeys(server.PublicKeys))
	}

	if len(metadata) > 0 {
		_ = d.Set("meta", metadata)
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

	if err := d.Set("enclave_page_caches", nativeServerEPC(server.EnclavePageCaches)); err != nil {
		return diag.Errorf("error setting Server EPC - error: %#v", err)
	}

	if err := d.Set("tags", nativeServerTags(server.Tags)); err != nil {
		return diag.Errorf("error setting Server tags - error: %#v", err)
	}

	return nil
}

func nativeServerIPv4(s *cloudsigmaapi.Server) string {
	if s.Runtime != nil {
		for _, n := range s.Runtime.RuntimeNICs {
			if n.InterfaceType == "public" {
				return n.IPv4.UUID
			}
		}
	}
	return ""
}
func nativeServerKeys(keys []cloudsigmaapi.Keypair) []interface{} {
	v := make([]interface{}, len(keys))
	for i, k := range keys {
		v[i] = k.UUID
	}
	return v
}
func nativeServerTags(tags []cloudsigmaapi.Tag) []interface{} {
	v := make([]interface{}, len(tags))
	for i, k := range tags {
		v[i] = k.UUID
	}
	return v
}
func nativeServerEPC(epcs []cloudsigmaapi.EnclavePageCache) []interface{} {
	v := make([]interface{}, len(epcs))
	for i, k := range epcs {
		v[i] = k.Size
	}
	return v
}
