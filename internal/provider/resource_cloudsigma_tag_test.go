package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func init() {
	resource.AddTestSweepers("cloudsigma_tag", &resource.Sweeper{
		Name: "cloudsigma_tag",
		F:    testSweepTags,
	})
}

func testSweepTags(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	tags, _, err := client.Tags.List(ctx)
	if err != nil {
		return fmt.Errorf("getting tags list: %s", err)
	}

	for _, tag := range tags {
		if strings.HasPrefix(tag.Name, accTestPrefix) {
			slog.Info("Deleting cloudsigma_tag", "name", tag.Name, "uuid", tag.UUID)
			_, err := client.Tags.Delete(ctx, tag.UUID)
			if err != nil {
				slog.Warn("Error deleting tag during sweep", "name", tag.Name, "error", err)
			}
		}
	}

	return nil
}

func TestAccResourceCloudSigmaTag_basic(t *testing.T) {
	var tag cloudsigma.Tag
	tagName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaTagResource(tagName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagExists("cloudsigma_tag.foobar", &tag),
					resource.TestCheckResourceAttr("cloudsigma_tag.foobar", "name", tagName),
					resource.TestCheckResourceAttrSet("cloudsigma_tag.foobar", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_tag.foobar", "resource_uri"),
				),
			},
		},
	})
}

func TestAccResourceCloudSigmaTag_update(t *testing.T) {
	var tag cloudsigma.Tag
	tagName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	tagNameUpdated := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaTagResource(tagName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagExists("cloudsigma_tag.foobar", &tag),
					resource.TestCheckResourceAttr("cloudsigma_tag.foobar", "name", tagName),
					resource.TestCheckResourceAttrSet("cloudsigma_tag.foobar", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_tag.foobar", "resource_uri"),
				),
			},
			{
				Config: testAccCloudSigmaTagResource(tagNameUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagExists("cloudsigma_tag.foobar", &tag),
					resource.TestCheckResourceAttr("cloudsigma_tag.foobar", "name", tagNameUpdated),
					resource.TestCheckResourceAttrSet("cloudsigma_tag.foobar", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_tag.foobar", "resource_uri"),
				),
			},
		},
	})
}

func TestAccResourceCloudSigmaTag_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaTagResourceWithoutName(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
		},
	})
}

func TestAccResourceCloudSigmaTag_upgradeFromSDK(t *testing.T) {
	tagName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckTagDestroy,

		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"cloudsigma": {
						VersionConstraint: "2.1.0",
						Source:            "cloudsigma/cloudsigma",
					},
				},
				Config: testAccCloudSigmaTagResource(tagName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsigma_tag.foobar", "name", tagName),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccCloudSigmaTagResource(tagName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccCheckTagDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sharedClient("testacc")
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudsigma_tag" {
			continue
		}

		tag, _, err := client.Tags.Get(ctx, rs.Primary.ID)
		if err == nil && tag.UUID == rs.Primary.ID {
			return fmt.Errorf("tag (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckTagExists(n string, tag *cloudsigma.Tag) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("no tag ID set")
		}

		ctx := context.Background()
		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		retrievedTag, _, err := client.Tags.Get(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("could not get tag: %s", err)
		}

		if retrievedTag.UUID != rs.Primary.ID {
			return errors.New("tag not found")
		}

		tag = retrievedTag
		return nil
	}
}

func testAccCloudSigmaTagResource(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_tag" "foobar" {
  name = "%s"
}`, name)
}

func testAccCloudSigmaTagResourceWithoutName() string {
	return `
resource "cloudsigma_tag" "foobar" {
}`
}
