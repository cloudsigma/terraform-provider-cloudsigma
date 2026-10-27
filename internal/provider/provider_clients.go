package provider

import (
	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

type authMode uint8

const (
	authBasic authMode = iota + 1
	authToken
)

// ProviderClients holds the explicit migration boundary. Native transport is
// available for both credential modes; individual provider surfaces opt in by
// auth mode while they migrate away from the SDK.
type ProviderClients struct {
	Native *cloudsigmaapi.Client
	Legacy *cloudsigma.Client
	Auth   authMode
}

func legacyClient(data interface{}) (*cloudsigma.Client, bool) {
	c, ok := data.(*ProviderClients)
	if !ok || c == nil || c.Legacy == nil || (c.Auth != authBasic && c.Auth != authToken) {
		return nil, false
	}
	return c.Legacy, true
}
