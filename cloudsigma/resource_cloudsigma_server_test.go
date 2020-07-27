package cloudsigma

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceCloudSigmaServer_Basic(t *testing.T) {
	serverCPU := 2000
	serverDrive := "044263cf-0a85-41f2-99b5-8d2a67294001"
	serverMemory := 1024
	serverName := fmt.Sprintf("server-%s", acctest.RandString(10))
	serverVNCPassword := fmt.Sprintf("vnc-%s", acctest.RandString(10))
	config := fmt.Sprintf(testAccResourceCloudSigmaServerConfig, serverCPU, serverDrive, serverMemory, serverName, serverVNCPassword)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Providers: testAccProviders,
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
  drive        = "%s"
  memory       = %d
  name         = "%s"
  vnc_password = "%s"
}
`
