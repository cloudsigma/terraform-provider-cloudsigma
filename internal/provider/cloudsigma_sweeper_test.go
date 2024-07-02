package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sharedClient(_ string) (*cloudsigma.Client, error) {
	location := os.Getenv("CLOUDSIGMA_LOCATION")
	if location == "" {
		return nil, fmt.Errorf("empty CLOUDSIGMA_LOCATION")
	}

	username := os.Getenv("CLOUDSIGMA_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("CLOUDSIGMA_USERNAME must be set for acceptance tests")
	}

	password := os.Getenv("CLOUDSIGMA_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("CLOUDSIGMA_PASSWORD must be set for acceptance tests")
	}

	opts := []cloudsigma.ClientOption{cloudsigma.WithUserAgent("terraform-provider-cloudsigma/sweeper")}
	opts = append(opts, cloudsigma.WithLocation(location))
	creds := cloudsigma.NewUsernamePasswordCredentialsProvider(username, password)
	return cloudsigma.NewClient(creds, opts...), nil
}
