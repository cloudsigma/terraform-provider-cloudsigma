package cloudsigma

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceCloudSigmaSSHKey_Basic(t *testing.T) {
	sshKeyName := fmt.Sprintf("key-%s", acctest.RandString(10))
	sshKeyPublic := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAA/yupp+bxv9EKJmg5LNwu1foNjby/Nl++Nx2XTmi80BRRa4daLNQYJ7oQ=="
	config := fmt.Sprintf(testAccResourceCloudSigmaSSHKeyConfig, sshKeyName, sshKeyPublic)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "name", sshKeyName),
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "public_key", sshKeyPublic),
				),
			},
		},
	})
}

const testAccResourceCloudSigmaSSHKeyConfig = `
resource "cloudsigma_ssh_key" "foobar" {
	name = "%s"
	public_key = "%s"
}
`
