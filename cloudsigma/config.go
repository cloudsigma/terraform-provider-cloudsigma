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
	var client *cloudsigma.Client
	if len(c.Token) > 0 {
		client = cloudsigma.NewTokenClient(c.Token, nil)
		log.Printf("[INFO] CloudSigma Client configured using access token, location: %s", c.Location)
	} else {
		client = cloudsigma.NewBasicAuthClient(c.Username, c.Password, nil)
		log.Printf("[INFO] CloudSigma Client configured for user: %s, location: %s", c.Username, c.Location)
	}
	client.SetAPIEndpoint(c.Location, c.BaseURL)
	client.SetUserAgent(c.userAgent)

	return client
}

// loadAndValidate configures and returns a fully initialized CloudSigma SDK.
func (c *Config) loadAndValidate(ctx context.Context, terraformVersion string) {
	c.context = ctx

	providerVersion := fmt.Sprintf("terraform-provider-cloudsigma/%s", providerVersion)
	userAgent := fmt.Sprintf("Terraform/%s (https://www.terraform.io) %s", terraformVersion, providerVersion)
	c.userAgent = userAgent
}
