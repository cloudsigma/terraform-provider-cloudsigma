package cloudsigma

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCloudSigmaServer_basic(t *testing.T) {
	var server cloudsigma.Server
	serverName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	tagName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaServerConfig_basic(serverName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "cpu", "2000"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "memory", "536870912"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "name", serverName),
					resource.TestCheckResourceAttrSet("cloudsigma_server.test", "resource_uri"),
				),
			},
			{
				Config: testAccCloudSigmaServerConfig_addTag(tagName, serverName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "tags.#", "1"),
				),
			},
			{
				Config: testAccCloudSigmaServerConfig_noTag(serverName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccCloudSigmaServer_emptySSH(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaServerConfig_emptySSHKey(),
				ExpectError: regexp.MustCompile("ssh_keys.* must not be empty, got"),
			},
		},
	})
}

func TestAccCloudSigmaServer_emptyTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaServerConfig_emptyTag(),
				ExpectError: regexp.MustCompile("tags.* must not be empty, got"),
			},
		},
	})
}

func TestAccCloudSigmaServer_smp(t *testing.T) {
	var server cloudsigma.Server
	serverName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaServerConfig_basic(serverName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "cpu", "2000"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "memory", "536870912"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "name", serverName),
					resource.TestCheckResourceAttrSet("cloudsigma_server.test", "resource_uri"),
				),
			},
			{
				Config: testAccCloudSigmaServerConfig_addSMP(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "smp", "2"),
				),
			},
		},
	})
}

func TestAccCloudSigmaServer_invalidSMP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaServerConfig_invalidSMP(),
				ExpectError: regexp.MustCompile("the minimum amount of cpu per smp is .*"),
			},
		},
	})
}

func TestAccCloudSigmaServer_withDrive(t *testing.T) {
	var server cloudsigma.Server
	var drive cloudsigma.Drive
	serverName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	driveName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaServerConfig_withDrive(serverName, driveName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "cpu", "2000"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "memory", "536870912"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "name", serverName),
					resource.TestCheckResourceAttrSet("cloudsigma_server.test", "resource_uri"),

					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "name", driveName),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "size", "5368709120"),
				),
			},
			{
				Config: testAccCloudSigmaServerConfig_changeDriveSize(serverName, driveName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),

					testAccCheckCloudSigmaDriveExists("cloudsigma_drive.test", &drive),
					resource.TestCheckResourceAttr("cloudsigma_drive.test", "size", "16106127360"),
				),
			},
		},
	})
}

func TestAccCloudSigmaServer_withMeta(t *testing.T) {
	var server cloudsigma.Server
	serverName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudSigmaServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaServerConfig_withMeta(serverName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "meta.%", "2"),
				),
			},
			{
				Config: testAccCloudSigmaServerConfig_changeMeta(serverName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCloudSigmaServerExists("cloudsigma_server.test", &server),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "meta.%", "3"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "meta.base64_fields", "cloudinit-user-data"),
					resource.TestCheckResourceAttr("cloudsigma_server.test", "meta.random-key", "random-value"),
				),
			},
		},
	})
}

func testAccCheckCloudSigmaServerDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*cloudsigma.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudsigma_server" {
			continue
		}

		server, _, err := client.Servers.Get(context.Background(), rs.Primary.ID)
		if err == nil && server.UUID == rs.Primary.ID {
			return fmt.Errorf("server (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckCloudSigmaServerExists(n string, server *cloudsigma.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no server ID is set")
		}

		client := testAccProvider.Meta().(*cloudsigma.Client)
		retrievedServer, _, err := client.Servers.Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if retrievedServer.UUID != rs.Primary.ID {
			return fmt.Errorf("server not found")
		}

		*server = *retrievedServer
		return nil
	}
}

func testAccCloudSigmaServerConfig_basic(serverName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"
}
`, serverName)
}

func testAccCloudSigmaServerConfig_addTag(tagName, serverName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_tag" "test" {
  name = "%s"
}

resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"

  tags = [cloudsigma_tag.test.id]
}
`, tagName, serverName)
}

func testAccCloudSigmaServerConfig_noTag(serverName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"

  tags = []
}
`, serverName)
}

func testAccCloudSigmaServerConfig_emptySSHKey() string {
	return `
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "server-with-invalid-empty-ssh-key-element"
  vnc_password = "cloudsigma"

  ssh_keys = [""]
}
`
}

func testAccCloudSigmaServerConfig_emptyTag() string {
	return `
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "server-with-invalid-empty-tag-element"
  vnc_password = "cloudsigma"

  tags = [""]
}
`
}

func testAccCloudSigmaServerConfig_addSMP() string {
	return `
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "server-with-invalid-empty-tag-element"
  smp          = 2
  vnc_password = "cloudsigma"
}
`
}

func testAccCloudSigmaServerConfig_invalidSMP() string {
	return `
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "server-with-invalid-empty-tag-element"
  smp          = 5
  vnc_password = "cloudsigma"
}
`
}

func testAccCloudSigmaServerConfig_withDrive(serverName, driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"
}

resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 5 * 1024 * 1024 * 1024
}
`, serverName, driveName)
}

func testAccCloudSigmaServerConfig_changeDriveSize(serverName, driveName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"
}

resource "cloudsigma_drive" "test" {
  media = "disk"
  name  = "%s"
  size  = 15 * 1024 * 1024 * 1024
}
`, serverName, driveName)
}

func testAccCloudSigmaServerConfig_withMeta(serverName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"

  meta = {
    base64_fields = "cloudinit-user-data"
    cloudinit-user-data = "I2Nsb3VkLWNvbmZpZw=="
  }
}
`, serverName)
}

func testAccCloudSigmaServerConfig_changeMeta(serverName string) string {
	return fmt.Sprintf(`
resource "cloudsigma_server" "test" {
  cpu          = 2000
  memory       = 536870912
  name         = "%s"
  vnc_password = "cloudsigma"

  meta = {
    base64_fields = "cloudinit-user-data"
    cloudinit-user-data = "I2Nsb3VkLWNvbmZpZw=="
    random-key = "random-value"
  }
}
`, serverName)
}

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
