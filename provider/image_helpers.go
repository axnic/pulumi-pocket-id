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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"path"

	"github.com/pulumi/pulumi-go-provider/infer/types"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

// imageUpload is the content of an image input, ready to be sent.
type imageUpload struct {
	data     []byte
	filename string
	sha256   string
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// isEmptyImage reports whether no asset was provided.
func isEmptyImage(img types.AssetOrArchive) bool { return img.Asset == nil && img.Archive == nil }

// readImage loads the bytes of an image input. Only assets are accepted (a
// FileAsset, StringAsset or RemoteAsset); archives are rejected.
func readImage(img types.AssetOrArchive) (imageUpload, error) {
	if img.Archive != nil {
		return imageUpload{}, errors.New("image must be an asset, not an archive")
	}
	if img.Asset == nil {
		return imageUpload{}, errors.New("image is required")
	}
	data, err := img.Asset.Bytes()
	if err != nil {
		return imageUpload{}, err
	}
	name := ""
	switch {
	case img.Asset.IsPath():
		name = img.Asset.Path
	case img.Asset.IsURI():
		if u, err := url.Parse(img.Asset.URI); err == nil {
			name = u.Path
		}
	}
	return imageUpload{
		data: data, filename: pocketidclient.ImageFilename(path.Base(name), data), sha256: sha256Hex(data),
	}, nil
}

// imageSHA256 returns the digest of an image input. The engine may hand the
// provider an asset reduced to its hash (its contents are not always
// available, e.g. in diffs); for assets that hash is the SHA-256 of the
// content, so it is used as is. "" means the digest cannot be known (e.g. a
// file that does not exist yet during a preview).
func imageSHA256(img types.AssetOrArchive) string {
	if up, err := readImage(img); err == nil {
		return up.sha256
	}
	if img.Asset != nil {
		return img.Asset.Hash
	}
	return ""
}
