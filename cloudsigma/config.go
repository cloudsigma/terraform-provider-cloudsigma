package cloudsigma

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Config represents the configuration structure used to instantiate
// the Cloudsigma provider.
type Config struct {
	Token    string
	Username string
	Password string
	Location string
	BaseURL  string

	context   context.Context
	userAgent string
}

// Client returns a new client for accessing CloudSigma.
func (c *Config) Client() *cloudsigma.Client {
	var creds cloudsigma.CredentialsProvider
	if len(c.Token) > 0 {
		creds = cloudsigma.NewTokenCredentialsProvider(c.Token)
		tflog.Info(c.context, "CloudSigma Client configured using access token", map[string]interface{}{
			"location": c.Location,
		})
	} else {
		creds = cloudsigma.NewUsernamePasswordCredentialsProvider(c.Username, c.Password)
		tflog.Info(c.context, "CloudSigma Client configured for user", map[string]interface{}{
			"location": c.Location,
			"username": c.Username,
		})
		log.Printf("[INFO] CloudSigma Client configured for user: %s, location: %s", c.Username, c.Location)
	}
	client := cloudsigma.NewClient(
		creds,
		cloudsigma.WithLocation(c.Location), cloudsigma.WithUserAgent(c.userAgent),
	)

	return client
}

// loadAndValidate configures and returns a fully initialized CloudSigma SDK.
func (c *Config) loadAndValidate(ctx context.Context, terraformVersion string) {
	c.context = ctx

	providerVersion := fmt.Sprintf("terraform-provider-cloudsigma/%s", providerVersion)
	userAgent := fmt.Sprintf("Terraform/%s (https://www.terraform.io) %s", terraformVersion, providerVersion)
	c.userAgent = userAgent
}
