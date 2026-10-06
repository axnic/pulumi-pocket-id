// Copyright 2025, axnic.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

// clientFromContext returns the API client built by Config.Configure.
func clientFromContext(ctx context.Context) *pocketidclient.Client {
	return infer.GetConfig[Config](ctx).client
}

// isGone reports whether err means the remote object no longer exists. Read
// methods use it to return an empty ID (resource dropped from state, i.e.
// drift) and Delete methods to stay idempotent.
func isGone(err error) bool { return pocketidclient.IsNotFound(err) }

// Resources and functions register themselves from an init() in their own
// file, so adding one never requires editing provider.go:
//
//	func init() { registerResource(infer.Resource(&Foo{})) }
var (
	registeredResources []infer.InferredResource
	registeredFunctions []infer.InferredFunction
)

func registerResource(r infer.InferredResource) { registeredResources = append(registeredResources, r) }
func registerFunction(f infer.InferredFunction) { registeredFunctions = append(registeredFunctions, f) }
