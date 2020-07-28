package cloudsigma

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceCloudSigmaTag_Basic(t *testing.T) {
	tagName := fmt.Sprintf("tag-%s", acctest.RandString(10))
	config := fmt.Sprintf(testAccResourceCloudSigmaTagConfig, tagName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsigma_tag.foobar", "name", tagName),
				),
			},
		},
	})
}

const testAccResourceCloudSigmaTagConfig = `
resource "cloudsigma_tag" "foobar" {
  name = "%s"
}
`
