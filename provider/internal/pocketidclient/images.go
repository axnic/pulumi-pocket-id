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

package pocketidclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

// Application image kinds, as accepted by the application-images endpoints.
const (
	ImageLogo                  = "logo"
	ImageDarkLogo              = "darkLogo"
	ImageFavicon               = "favicon"
	ImageBackground            = "background"
	ImageEmail                 = "email"
	ImageDefaultProfilePicture = "defaultProfilePicture"
)

// ImageKinds lists every application image kind.
var ImageKinds = []string{
	ImageLogo, ImageDarkLogo, ImageFavicon, ImageBackground, ImageEmail, ImageDefaultProfilePicture,
}

// imageExtensions maps detected content types to the file extension Pocket-ID
// uses to validate uploads.
var imageExtensions = map[string]string{
	"image/png":                ".png",
	"image/jpeg":               ".jpg",
	"image/svg+xml":            ".svg",
	"image/webp":               ".webp",
	"image/gif":                ".gif",
	"image/x-icon":             ".ico",
	"image/vnd.microsoft.icon": ".ico",
}

// ImageFilename returns the upload file name: name when it already has a known
// image extension, otherwise "image" with the extension matching the content.
func ImageFilename(name string, data []byte) string {
	ext := strings.ToLower(path.Ext(name))
	for _, known := range imageExtensions {
		if ext == known || ext == ".jpeg" {
			return path.Base(name)
		}
	}
	if ext := imageExtensions[DetectContentType(data)]; ext != "" {
		return "image" + ext
	}
	return "image"
}

// applicationImageEndpoint maps an image kind to its path, query and whether
// the API can delete it (favicon and e-mail images cannot be deleted).
func applicationImageEndpoint(kind string) (p string, q url.Values, deletable bool, err error) {
	const base = "/api/application-images/"
	switch kind {
	case ImageLogo:
		return base + "logo", lightQuery(true), true, nil
	case ImageDarkLogo:
		return base + "logo", lightQuery(false), true, nil
	case ImageFavicon:
		return base + "favicon", nil, false, nil
	case ImageBackground:
		return base + "background", nil, true, nil
	case ImageEmail:
		return base + "email", nil, false, nil
	case ImageDefaultProfilePicture:
		return base + "default-profile-picture", nil, true, nil
	}
	return "", nil, false, fmt.Errorf("unknown application image type %q", kind)
}

// CanDeleteApplicationImage reports whether the API can delete the image kind.
func CanDeleteApplicationImage(kind string) bool {
	_, _, ok, err := applicationImageEndpoint(kind)
	return err == nil && ok
}

// SetApplicationImage uploads (or replaces) an application image.
func (c *Client) SetApplicationImage(ctx context.Context, kind, filename string, data []byte) error {
	p, q, _, err := applicationImageEndpoint(kind)
	if err != nil {
		return err
	}
	return c.DoMultipart(ctx, http.MethodPut, p, q, filename, data)
}

// GetApplicationImage returns the raw bytes of the custom image. The bundled
// default is not returned where the API allows telling them apart, so a
// missing custom image is a 404 on logos.
func (c *Client) GetApplicationImage(ctx context.Context, kind string) ([]byte, error) {
	p, q, _, err := applicationImageEndpoint(kind)
	if err != nil {
		return nil, err
	}
	if kind == ImageLogo || kind == ImageDarkLogo {
		q.Set("default", "false")
	}
	return c.GetBytes(ctx, p, q)
}

// DeleteApplicationImage deletes the custom image and restores the default.
// It fails for kinds the API cannot delete; see CanDeleteApplicationImage.
func (c *Client) DeleteApplicationImage(ctx context.Context, kind string) error {
	p, q, ok, err := applicationImageEndpoint(kind)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("application image %q cannot be deleted", kind)
	}
	return c.Do(ctx, http.MethodDelete, p, q, nil, nil)
}

func oidcClientLogoPath(clientID string) string {
	return "/api/oidc/clients/" + url.PathEscape(clientID) + "/logo"
}

func lightQuery(light bool) url.Values { return url.Values{"light": {strconv.FormatBool(light)}} }

// SetOidcClientLogo uploads the light or dark logo of an OIDC client.
func (c *Client) SetOidcClientLogo(
	ctx context.Context, clientID string, light bool, filename string, data []byte,
) error {
	return c.DoMultipart(ctx, http.MethodPost, oidcClientLogoPath(clientID), lightQuery(light), filename, data)
}

// GetOidcClientLogo returns the raw bytes of the light or dark logo of an OIDC client.
func (c *Client) GetOidcClientLogo(ctx context.Context, clientID string, light bool) ([]byte, error) {
	return c.GetBytes(ctx, oidcClientLogoPath(clientID), lightQuery(light))
}

// DeleteOidcClientLogo removes the light or dark logo of an OIDC client.
func (c *Client) DeleteOidcClientLogo(ctx context.Context, clientID string, light bool) error {
	return c.Do(ctx, http.MethodDelete, oidcClientLogoPath(clientID), lightQuery(light), nil, nil)
}
