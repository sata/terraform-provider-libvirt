package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	libvirtclient "github.com/dmacvicar/terraform-provider-libvirt/v2/internal/libvirt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	resource.AddTestSweepers("libvirt_nwfilter", &resource.Sweeper{
		Name: "libvirt_nwfilter",
		F: func(uri string) error {
			ctx := context.Background()
			client, err := libvirtclient.NewClient(ctx, uri)
			if err != nil {
				return fmt.Errorf("failed to create libvirt client: %w", err)
			}
			defer func() { _ = client.Close() }()

			filters, _, err := client.Libvirt().ConnectListAllNwfilters(1, 0)
			if err != nil {
				return fmt.Errorf("failed to list nwfilters: %w", err)
			}

			for _, filter := range filters {
				if strings.HasPrefix(filter.Name, "test-") || strings.HasPrefix(filter.Name, "test_") {
					if err := client.Libvirt().NwfilterUndefine(filter); err != nil {
						fmt.Printf("Warning: failed to undefine nwfilter %s: %v\n", filter.Name, err)
					}
				}
			}

			return nil
		},
	})
}

func testAccCheckNWFilterDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := libvirtclient.NewClient(ctx, testAccLibvirtURI())
	if err != nil {
		return fmt.Errorf("failed to create libvirt client: %w", err)
	}
	defer func() { _ = client.Close() }()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "libvirt_nwfilter" {
			continue
		}

		uuid := rs.Primary.Attributes["uuid"]
		if uuid == "" {
			continue
		}

		_, err := client.LookupNWFilterByUUID(uuid)
		if err == nil {
			return fmt.Errorf("nwfilter %s still exists after destroy", uuid)
		}
	}

	return nil
}

func TestAccNWFilterResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNWFilterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNWFilterConfigBasic("test-nwfilter-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("libvirt_nwfilter.test", "name", "test-nwfilter-basic"),
					resource.TestCheckResourceAttrSet("libvirt_nwfilter.test", "uuid"),
					resource.TestCheckResourceAttrSet("libvirt_nwfilter.test", "id"),
				),
			},
		},
	})
}

func TestAccNWFilterResource_withRef(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNWFilterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNWFilterConfigWithRef("test-nwfilter-ref"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("libvirt_nwfilter.test", "name", "test-nwfilter-ref"),
					resource.TestCheckResourceAttrSet("libvirt_nwfilter.test", "uuid"),
				),
			},
		},
	})
}

func TestAccNWFilterResource_withRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNWFilterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNWFilterConfigWithRule("test-nwfilter-rule"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("libvirt_nwfilter.test", "name", "test-nwfilter-rule"),
					resource.TestCheckResourceAttrSet("libvirt_nwfilter.test", "uuid"),
				),
			},
		},
	})
}

func TestAccNWFilterResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNWFilterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNWFilterConfigBasic("test-nwfilter-import"),
			},
			{
				ResourceName:      "libvirt_nwfilter.test",
				ImportState:       true,
				ImportStateVerify: false, // name-based import sets id; Read takes over
				ImportStateId:     "test-nwfilter-import",
			},
		},
	})
}

func testAccNWFilterConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "libvirt_nwfilter" "test" {
  name = %q
}
`, name)
}

func testAccNWFilterConfigWithRef(name string) string {
	return fmt.Sprintf(`
resource "libvirt_nwfilter" "test" {
  name = %q
  entries = [
    {
      ref = {
        filter = "clean-traffic"
        parameters = [
          { name = "IP",  value = "192.168.122.10" },
          { name = "MAC", value = "52:54:00:12:34:56" },
        ]
      }
    }
  ]
}
`, name)
}

func testAccNWFilterConfigWithRule(name string) string {
	return fmt.Sprintf(`
resource "libvirt_nwfilter" "test" {
  name = %q
  entries = [
    {
      rule = {
        action    = "accept"
        direction = "in"
        tcp = {
          dst_port_start = "22"
          dst_port_end   = "22"
        }
      }
    },
    {
      rule = {
        action    = "drop"
        direction = "in"
      }
    },
  ]
}
`, name)
}
