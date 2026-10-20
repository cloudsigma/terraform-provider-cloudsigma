package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// configClients keeps SDKv2 token precedence separate from Framework auth policy.
// Only the Drive resource Read uses native; mutations and waiters retain legacy.
type configClients struct {
	legacy               *cloudsigma.Client
	native               *cloudsigmaapi.Client
	basicCredentialError error
}

func (c *Config) clients() (*configClients, error) {
	clients := &configClients{legacy: c.Client()}
	if c.Token == "" {
		// Match SDKv2 request-time rejection without retaining another credential copy.
		if c.Username == "" {
			clients.basicCredentialError = errors.New("username must not be empty")
		} else if c.Password == "" {
			clients.basicCredentialError = errors.New("password must not be empty")
		}
		var err error
		clients.native, err = cloudsigmaapi.New(c.Location, c.Username, c.Password, c.userAgent)
		if err != nil {
			return nil, err
		}
	}
	return clients, nil
}

func extractLegacy(meta interface{}) (*cloudsigma.Client, error) {
	switch c := meta.(type) {
	case *cloudsigma.Client:
		if c != nil {
			return c, nil
		}
	case *configClients:
		if c != nil && c.legacy != nil {
			return c.legacy, nil
		}
	}
	return nil, errors.New("CloudSigma provider client is not configured")
}

func readNativeDrive(ctx context.Context, d *schema.ResourceData, client *cloudsigmaapi.Client) diag.Diagnostics {
	if err := ctx.Err(); err != nil {
		return diag.FromErr(err)
	}
	drive, err := client.Drive(ctx, d.Id())
	// A cancelled refresh must not remove state even if a late response was 404.
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
	_ = d.Set("media", drive.Media)
	_ = d.Set("name", drive.Name)
	_ = d.Set("resource_uri", drive.ResourceURI)
	_ = d.Set("size", drive.Size)
	_ = d.Set("status", drive.Status)
	_ = d.Set("storage_type", drive.StorageType)
	_ = d.Set("uuid", drive.UUID)
	mounted := make([]interface{}, len(drive.MountedOn))
	for i, link := range drive.MountedOn {
		mounted[i] = map[string]interface{}{"resource_uri": link.ResourceURI, "uuid": link.UUID}
	}
	if err := d.Set("mounted_on", mounted); err != nil {
		return diag.FromErr(err)
	}
	tags := make([]interface{}, len(drive.Tags))
	for i, tag := range drive.Tags {
		tags[i] = tag.UUID
	}
	if err := d.Set("tags", tags); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
