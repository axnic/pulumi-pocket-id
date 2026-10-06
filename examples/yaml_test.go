//go:build yaml || all
// +build yaml all

package examples

import (
	"os"
	"testing"

	"github.com/pulumi/providertest/pulumitest"
	"github.com/pulumi/providertest/pulumitest/opttest"
)

// TestYAMLExampleLifecycle drives the yaml example program's full
// create/read/update/delete lifecycle against a live Pocket-ID instance. It
// needs POCKET_ID_BASE_URL/POCKET_ID_API_KEY pointing at one - see
// `make test_e2e` (Docker) or CONTRIBUTING.md for how to run one locally.
func TestYAMLExampleLifecycle(t *testing.T) {
	requirePocketID(t)

	pt := pulumitest.NewPulumiTest(t, "yaml",
		opttest.AttachProviderServer("pocket-id", providerFactory),
		opttest.SkipInstall(),
	)

	pt.SetConfig(t, "pocket-id:baseUrl", os.Getenv("POCKET_ID_BASE_URL"))
	pt.SetConfig(t, "pocket-id:apiKey", os.Getenv("POCKET_ID_API_KEY"))

	pt.Preview(t)
	pt.Up(t)
	pt.Destroy(t)
}
