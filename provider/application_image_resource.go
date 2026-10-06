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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/infer/types"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&ApplicationImage{})) }

// ApplicationImageType selects which Pocket-ID image an ApplicationImage manages.
type ApplicationImageType string

// Values lists the accepted image types.
func (ApplicationImageType) Values() []infer.EnumValue[ApplicationImageType] {
	return []infer.EnumValue[ApplicationImageType]{
		{Name: "Logo", Value: pocketidclient.ImageLogo, Description: "The light mode application logo."},
		{Name: "DarkLogo", Value: pocketidclient.ImageDarkLogo, Description: "The dark mode application logo."},
		{Name: "Favicon", Value: pocketidclient.ImageFavicon, Description: "The favicon. It cannot be deleted."},
		{Name: "Background", Value: pocketidclient.ImageBackground, Description: "The login page background."},
		{Name: "Email", Value: pocketidclient.ImageEmail, Description: "The image used in e-mails. It cannot be deleted."},
		{Name: "DefaultProfilePicture", Value: pocketidclient.ImageDefaultProfilePicture,
			Description: "The default profile picture of users."},
	}
}

// ApplicationImage uploads one of the global Pocket-ID branding images.
type ApplicationImage struct{}

// ApplicationImageArgs are the inputs of an ApplicationImage.
type ApplicationImageArgs struct {
	Type  ApplicationImageType `pulumi:"type" provider:"replaceOnChanges"`
	Image types.AssetOrArchive `pulumi:"image"`
}

// ApplicationImageState is the persisted state of an ApplicationImage.
type ApplicationImageState struct {
	ApplicationImageArgs
	SHA256 string `pulumi:"sha256"`
}

var _ infer.Annotated = (*ApplicationImageArgs)(nil)

// Annotate describes the inputs of an ApplicationImage.
func (a *ApplicationImageArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Type, "Which image to manage. Changing it replaces the resource.")
	an.Describe(&a.Image, "The image file as an asset, e.g. a FileAsset. Archives are not supported. "+
		"A change of content re-uploads the file. It is not read back as an asset: after an import "+
		"it is empty and must be set in the program.")
}

var _ infer.Annotated = (*ApplicationImageState)(nil)

// Annotate describes the outputs of an ApplicationImage.
func (s *ApplicationImageState) Annotate(an infer.Annotator) {
	an.Describe(&s.SHA256, "The hex SHA-256 digest of the image content. At refresh it is the digest of the "+
		"bytes currently served by Pocket-ID, so out-of-band changes show up as a diff.")
}

var _ infer.Annotated = (*ApplicationImage)(nil)

// Annotate describes the ApplicationImage resource.
func (*ApplicationImage) Annotate(an infer.Annotator) {
	an.Describe(&ApplicationImage{}, "A global Pocket-ID branding image (logo, dark logo, favicon, background, "+
		"e-mail image or default profile picture) uploaded from a local file or asset. Destroying it deletes "+
		"the custom image and restores the default, except for the favicon and the e-mail image, which the API "+
		"cannot delete: destroying those only removes the resource from state. Import with the type, e.g. "+
		"\"favicon\". Per-user profile pictures are not covered.")
}

func uploadApplicationImage(ctx context.Context, args ApplicationImageArgs) (string, error) {
	up, err := readImage(args.Image)
	if err != nil {
		return "", err
	}
	err = clientFromContext(ctx).SetApplicationImage(ctx, string(args.Type), up.filename, up.data)
	return up.sha256, err
}

// Create uploads the image.
func (ApplicationImage) Create(
	ctx context.Context, req infer.CreateRequest[ApplicationImageArgs],
) (infer.CreateResponse[ApplicationImageState], error) {
	if req.DryRun {
		return infer.CreateResponse[ApplicationImageState]{
			ID: string(req.Inputs.Type), Output: ApplicationImageState{
				ApplicationImageArgs: req.Inputs, SHA256: imageSHA256(req.Inputs.Image),
			},
		}, nil
	}
	sum, err := uploadApplicationImage(ctx, req.Inputs)
	if err != nil {
		return infer.CreateResponse[ApplicationImageState]{}, err
	}
	return infer.CreateResponse[ApplicationImageState]{
		ID: string(req.Inputs.Type), Output: ApplicationImageState{ApplicationImageArgs: req.Inputs, SHA256: sum},
	}, nil
}

// Read downloads the image and records its digest; a missing image drops the resource from state.
func (ApplicationImage) Read(
	ctx context.Context, req infer.ReadRequest[ApplicationImageArgs, ApplicationImageState],
) (infer.ReadResponse[ApplicationImageArgs, ApplicationImageState], error) {
	args := ApplicationImageArgs{Type: ApplicationImageType(req.ID), Image: req.State.Image}
	if isEmptyImage(args.Image) {
		args.Image = req.Inputs.Image
	}
	data, err := clientFromContext(ctx).GetApplicationImage(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[ApplicationImageArgs, ApplicationImageState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[ApplicationImageArgs, ApplicationImageState]{}, err
	}
	return infer.ReadResponse[ApplicationImageArgs, ApplicationImageState]{
		ID: req.ID, Inputs: args, State: ApplicationImageState{ApplicationImageArgs: args, SHA256: sha256Hex(data)},
	}, nil
}

// Diff replaces on type changes and updates when the content digest differs.
func (ApplicationImage) Diff(
	_ context.Context, req infer.DiffRequest[ApplicationImageArgs, ApplicationImageState],
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Type != req.State.Type {
		diff["type"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if imageSHA256(req.Inputs.Image) != req.State.SHA256 {
		diff["image"] = p.PropertyDiff{Kind: p.Update}
	}
	return p.DiffResponse{HasChanges: len(diff) > 0, DetailedDiff: diff}, nil
}

// Update re-uploads the image (the upload is idempotent).
func (ApplicationImage) Update(
	ctx context.Context, req infer.UpdateRequest[ApplicationImageArgs, ApplicationImageState],
) (infer.UpdateResponse[ApplicationImageState], error) {
	if req.DryRun {
		return infer.UpdateResponse[ApplicationImageState]{
			Output: ApplicationImageState{ApplicationImageArgs: req.Inputs, SHA256: imageSHA256(req.Inputs.Image)},
		}, nil
	}
	sum, err := uploadApplicationImage(ctx, req.Inputs)
	if err != nil {
		return infer.UpdateResponse[ApplicationImageState]{}, err
	}
	return infer.UpdateResponse[ApplicationImageState]{
		Output: ApplicationImageState{ApplicationImageArgs: req.Inputs, SHA256: sum},
	}, nil
}

// Delete deletes the custom image when the API allows it; favicon and e-mail
// images cannot be deleted, so for them it only drops the resource from state.
func (ApplicationImage) Delete(
	ctx context.Context, req infer.DeleteRequest[ApplicationImageState],
) (infer.DeleteResponse, error) {
	kind := string(req.State.Type)
	if !pocketidclient.CanDeleteApplicationImage(kind) {
		return infer.DeleteResponse{}, nil
	}
	if err := clientFromContext(ctx).DeleteApplicationImage(ctx, kind); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}
