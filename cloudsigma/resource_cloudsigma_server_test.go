package cloudsigma

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAccResourceCloudSigmaServer_Basic(t *testing.T) {
	var providers []*schema.Provider
	serverCPU := 2000
	serverMemory := 512 * 1024 * 1024
	serverName := fmt.Sprintf("server-%s", acctest.RandString(10))
	serverVNCPassword := fmt.Sprintf("vnc-%s", acctest.RandString(10))
	config := fmt.Sprintf(testAccResourceCloudSigmaServerConfig, serverCPU, serverMemory, serverName, serverVNCPassword)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(&providers),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsigma_server.foobar", "cpu", strconv.Itoa(serverCPU)),
					resource.TestCheckResourceAttr("cloudsigma_server.foobar", "memory", strconv.Itoa(serverMemory)),
					resource.TestCheckResourceAttr("cloudsigma_server.foobar", "name", serverName),
					resource.TestCheckResourceAttr("cloudsigma_server.foobar", "vnc_password", serverVNCPassword),
				),
			},
		},
	})
}

const testAccResourceCloudSigmaServerConfig = `
resource "cloudsigma_server" "foobar" {
	cpu          = %d
  memory       = %d
  name         = "%s"
  vnc_password = "%s"
}
`

func TestResourceCloudSigmaServer_findIPv4Address(t *testing.T) {
	cases := []struct {
		server      *cloudsigma.Server
		addressType string
		expected    string
	}{
		{&cloudsigma.Server{}, "public", ""},
		{&cloudsigma.Server{
			Runtime: &cloudsigma.ServerRuntime{
				RuntimeNICs: []cloudsigma.ServerRuntimeNIC{{
					InterfaceType: "private", IPv4: cloudsigma.ServerRuntimeIP{UUID: "10.1.1.1"},
				}},
			},
		}, "public", ""},
		{&cloudsigma.Server{
			Runtime: &cloudsigma.ServerRuntime{
				RuntimeNICs: []cloudsigma.ServerRuntimeNIC{{
					InterfaceType: "public", IPv4: cloudsigma.ServerRuntimeIP{UUID: "178.33.44.55"},
				}},
			},
		}, "public", "178.33.44.55"},
	}

	for _, c := range cases {
		got := findIPv4Address(c.server, c.addressType)
		if !reflect.DeepEqual(c.expected, got) {
			t.Fatalf("expected: %v, got: %v", c.expected, got)
		}
	}
}
