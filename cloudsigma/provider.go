package cloudsigma

import (
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Provider returns a schema.Provider for cloudsigma.
func Provider() terraform.ResourceProvider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSIGMA_USERNAME", os.Getenv("CLOUDSIGMA_USERNAME")),
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSIGMA_PASSWORD", os.Getenv("CLOUDSIGMA_PASSWORD")),
			},
			"location": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSIGMA_LOCATION", os.Getenv("CLOUDSIGMA_LOCATION")),
				Default:     "zrh",
				Description: "The location endpoint for CloudSigma. Default is 'zrh'.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			// "cloudsigma_capabilities": dataSourceCloudSigmaCapabilities(),
			"cloudsigma_cloud_status": dataSourceCloudSigmaCloudStatus(),
			"cloudsigma_license":      dataSourceCloudSigmaLicense(),
			"cloudsigma_location":     dataSourceCloudSigmaLocation(),
			"cloudsigma_profile":      dataSourceCloudSigmaProfile(),
			"cloudsigma_subscription": dataSourceCloudSigmaSubscription(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"cloudsigma_acl":             resourceCloudSigmaACL(),
			"cloudsigma_firewall_policy": resourceCloudSigmaFirewallPolicy(),
			"cloudsigma_server":          resourceCloudSigmaServer(),
			"cloudsigma_ssh_key":         resourceCloudSigmaSSHKey(),
			"cloudsigma_tag":             resourceCloudSigmaTag(),
		},
	}

	provider.ConfigureFunc = providerConfigure(provider)

	return provider
}

func providerConfigure(provider *schema.Provider) schema.ConfigureFunc {
	return func(d *schema.ResourceData) (interface{}, error) {
		config := &Config{
			Username: d.Get("username").(string),
			Password: d.Get("password").(string),
			Location: d.Get("location").(string),
		}

		config.loadAndValidate(provider.StopContext(), provider.TerraformVersion)

		return config.Client(), nil
	}
}
