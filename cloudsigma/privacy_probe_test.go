package cloudsigma

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

func TestOfflinePrivacyFixed(t *testing.T) {
	var b bytes.Buffer
	old := log.Writer()
	log.SetOutput(&b)
	defer log.SetOutput(old)
	c := Config{Username: "synthetic-private-email", Password: "synthetic-password", Location: "next"}
	c.loadAndValidate(context.Background(), "offline")
	c.Client()
	if strings.Contains(b.String(), "synthetic-private-email") {
		t.Fatal("username leaked")
	}
	if strings.Contains(b.String(), "synthetic-password") {
		t.Fatal("password leak")
	}
	if !resourceCloudSigmaServer().Schema["vnc_password"].Sensitive {
		t.Fatal("password not sensitive")
	}
}
