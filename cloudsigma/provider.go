package cloudsigma

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns a schema.Provider for CloudSigma.
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				DefaultFunc:   schema.EnvDefaultFunc("CLOUDSIGMA_TOKEN", nil),
				Description:   "The CloudSigma access token.",
				ConflictsWith: []string{"username", "password"},
			},
			"username": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("CLOUDSIGMA_USERNAME", nil),
				Description:   "The CloudSigma user email.",
				ConflictsWith: []string{"token"},
			},
			"password": {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				DefaultFunc:   schema.EnvDefaultFunc("CLOUDSIGMA_PASSWORD", nil),
				Description:   "The CloudSigma password.",
				ConflictsWith: []string{"token"},
			},
			"location": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSIGMA_LOCATION", "zrh"),
				Description: "The location endpoint for CloudSigma. Default is 'zrh'.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSIGMA_BASE_URL", "cloudsigma.com/api/2.0/"),
				Description: "The base URL endpoint for CloudSigma. Default is 'cloudsigma.com/api/2.0/'.",
				Deprecated: `This "base_url" attribute is unused and will be removed in a future version of the provider. ` +
					"Please use location to specify CloudSigma API endpoint if needed: https://docs.cloudsigma.com/en/latest/general.html#api-endpoint.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{},

		ResourcesMap: map[string]*schema.Resource{
			"cloudsigma_drive":           resourceCloudSigmaDrive(),
			"cloudsigma_remote_snapshot": resourceCloudSigmaRemoteSnapshot(),
			"cloudsigma_server":          resourceCloudSigmaServer(),
			"cloudsigma_snapshot":        resourceCloudSigmaSnapshot(),
		},
	}

	provider.ConfigureContextFunc = providerConfigure(provider)

	return provider
}

func providerConfigure(provider *schema.Provider) schema.ConfigureContextFunc {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config := &Config{
			Token:    d.Get("token").(string),
			Username: d.Get("username").(string),
			Password: d.Get("password").(string),
			Location: d.Get("location").(string),
			BaseURL:  d.Get("base_url").(string),
		}

		config.loadAndValidate(ctx, provider.TerraformVersion)

		return config.Client(), nil
	}
}
