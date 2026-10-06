package examples

import (
	"os"
	"testing"

	"github.com/pulumi/providertest/providers"
	goprovider "github.com/pulumi/pulumi-go-provider"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"

	"github.com/axnic/pulumi-pocket-id/provider"
)

var providerFactory = func(_ providers.PulumiTest) (pulumirpc.ResourceProviderServer, error) { //nolint:unused
	return goprovider.RawServer("pocket-id", "1.0.0", provider.Provider())(nil)
}

// requirePocketID skips the calling test unless POCKET_ID_BASE_URL is set,
// keeping `make test` fast/hermetic while still letting `make test_e2e` (or
// a developer with a local Pocket-ID running) exercise the full
// create/read/update/delete lifecycle against a real instance.
func requirePocketID(t *testing.T) { //nolint:unused
	t.Helper()
	if os.Getenv("POCKET_ID_BASE_URL") == "" {
		t.Skip("set POCKET_ID_BASE_URL (and POCKET_ID_API_KEY) to run this test against a live Pocket-ID instance; " +
			"see `make test_e2e`")
	}
}
