//go:build go || all
// +build go all

package examples

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pulumi/providertest/pulumitest"
	"github.com/pulumi/providertest/pulumitest/opttest"
	"github.com/stretchr/testify/require"
)

// TestGoExampleLifecycle drives the go example program's full
// create/read/update/delete lifecycle against a live Pocket-ID instance.
// Needs POCKET_ID_BASE_URL/POCKET_ID_API_KEY pointing at one - see
// `make test_e2e` (Docker) or CONTRIBUTING.md for how to run one locally.
func TestGoExampleLifecycle(t *testing.T) {
	requirePocketID(t)

	cwd, err := os.Getwd()
	require.NoError(t, err)

	module := filepath.Join(cwd, "../sdk/go/pulumi-pocket-id")
	pt := pulumitest.NewPulumiTest(t, "go",
		opttest.GoModReplacement("github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id", module),
		opttest.AttachProviderServer("pocket-id", providerFactory),
		opttest.SkipInstall(),
	)

	// The provider has no env var config fallback (unlike some Pulumi
	// providers) - baseUrl/apiKey must be set as explicit Pulumi config,
	// see README.md's Known limitations section.
	pt.SetConfig(t, "pocket-id:baseUrl", os.Getenv("POCKET_ID_BASE_URL"))
	pt.SetConfig(t, "pocket-id:apiKey", os.Getenv("POCKET_ID_API_KEY"))

	pt.Preview(t)
	pt.Up(t)
	pt.Destroy(t)
}
