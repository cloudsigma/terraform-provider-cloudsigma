package cloudsigma

import (
	"log"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type Config struct {
	Username string
	Password string
	Location string
}

// Client returns a new client for accessing CloudSigma.
func (c *Config) Client() *cloudsigma.Client {
	client := cloudsigma.NewBasicAuthClient(c.Username, c.Password)
	log.Printf("[INFO] CloudSigma Client configured for user: %s", c.Username)
	return client
}
