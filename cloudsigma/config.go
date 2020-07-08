package cloudsigma

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Config represents the configuration structure used to instantiate
// the Cloudsigma provider.
type Config struct {
	Username string
	Password string
	Location string

	context   context.Context
	userAgent string
}

// Client returns a new client for accessing CloudSigma.
func (c *Config) Client() *cloudsigma.Client {
	client := cloudsigma.NewBasicAuthClient(c.Username, c.Password)
	client.SetLocation(c.Location)
	client.SetUserAgent(c.userAgent)

	log.Printf("[INFO] CloudSigma Client configured for user: %s, location: %s", c.Username, c.Location)
	return client
}

// loadAndValidate configures and returns a fully initialized CloudSigma SDK.
func (c *Config) loadAndValidate(ctx context.Context, terraformVersion string) {
	c.context = ctx

	providerVersion := fmt.Sprintf("terraform-provider-cloudsigma/%s", providerVersion)
	userAgent := fmt.Sprintf("Terraform/%s (https://www.terraform.io) %s", terraformVersion, providerVersion)
	c.userAgent = userAgent
}
