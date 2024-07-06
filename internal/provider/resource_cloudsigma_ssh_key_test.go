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
	resource.AddTestSweepers("cloudsigma_ssh_key", &resource.Sweeper{
		Name: "cloudsigma_ssh_key",
		F:    testSweepSSHKeys,
	})
}

func testSweepSSHKeys(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	keypairs, _, err := client.Keypairs.List(ctx)
	if err != nil {
		return fmt.Errorf("getting SSH keys list: %s", err)
	}

	for _, keypair := range keypairs {
		if strings.HasPrefix(keypair.Name, accTestPrefix) {
			slog.Info("Deleting cloudsigma_ssh_key", "name", keypair.Name, "uuid", keypair.UUID)
			_, err := client.Keypairs.Delete(ctx, keypair.UUID)
			if err != nil {
				slog.Warn("Error deleting SSH key during sweep", "name", keypair.Name, "error", err)
			}
		}
	}

	return nil
}

func TestAccResourceCloudSigmaSSHKey_basic(t *testing.T) {
	var sshKey cloudsigma.Keypair
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaSSHKeyResource(sshKeyName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("cloudsigma_ssh_key.foobar", &sshKey),
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "name", sshKeyName),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "private_key"),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "public_key"),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "uuid"),
				),
			},
		},
	})
}

func TestAccResourceCloudSigmaSSHKey_update(t *testing.T) {
	var sshKey cloudsigma.Keypair
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyNameUpdated := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic, _, err := acctest.RandSSHKeyPair("cloudsigma@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("could not generate test SSH key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudSigmaSSHKeyResourceWithPublicKey(sshKeyName, sshKeyPublic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("cloudsigma_ssh_key.foobar", &sshKey),
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "name", sshKeyName),
					// private_key is empty when only public material is defined
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "private_key", ""),
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "public_key", sshKeyPublic),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "uuid"),
				),
			},
			{
				Config: testAccCloudSigmaSSHKeyResourceWithPublicKey(sshKeyNameUpdated, sshKeyPublic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("cloudsigma_ssh_key.foobar", &sshKey),
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "name", sshKeyNameUpdated),
					// private_key is empty when only public material is defined
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "private_key", ""),
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "public_key", sshKeyPublic),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "id"),
					resource.TestCheckResourceAttrSet("cloudsigma_ssh_key.foobar", "uuid"),
				),
			},
		},
	})
}

func TestAccResourceCloudSigmaSSHKey_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config:      testAccCloudSigmaSSHKeyResourceWithoutName(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
		},
	})
}

func TestAccResourceCloudSigmaSSHKey_upgradeFromSDK(t *testing.T) {
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"cloudsigma": {
						VersionConstraint: "2.1.0",
						Source:            "cloudsigma/cloudsigma",
					},
				},
				Config: testAccCloudSigmaSSHKeyResource(sshKeyName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsigma_ssh_key.foobar", "name", sshKeyName),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccCloudSigmaSSHKeyResource(sshKeyName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccCheckSSHKeyDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sharedClient("testacc")
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudsigma_ssh_key" {
			continue
		}

		sshKey, _, err := client.Keypairs.Get(ctx, rs.Primary.ID)
		if err == nil && sshKey.UUID == rs.Primary.ID {
			return fmt.Errorf("SSH key (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckSSHKeyExists(n string, sshKey *cloudsigma.Keypair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("no SSH key ID set")
		}

		ctx := context.Background()
		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		retrievedSSHKey, _, err := client.Keypairs.Get(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("could not get SSH key: %s", err)
		}

		if retrievedSSHKey.UUID != rs.Primary.ID {
			return errors.New("SSH key not found")
		}

		sshKey = retrievedSSHKey
		return nil
	}
}

func testAccCloudSigmaSSHKeyResource(name string) string {
	return fmt.Sprintf(`
resource "cloudsigma_ssh_key" "foobar" {
  name = "%s"
}`, name)
}

func testAccCloudSigmaSSHKeyResourceWithPublicKey(name, publicKey string) string {
	return fmt.Sprintf(`
resource "cloudsigma_ssh_key" "foobar" {
  name = "%s"
  public_key = "%s"
}`, name, publicKey)
}

func testAccCloudSigmaSSHKeyResourceWithoutName() string {
	return `
resource "cloudsigma_ssh_key" "foobar" {
}`
}
