package main

import (
	pocketid "github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		group, err := pocketid.NewUserGroup(ctx, "myGroup", &pocketid.UserGroupArgs{
			Name:         pulumi.String("my-app-group"),
			FriendlyName: pulumi.String("My Application Group"),
		})
		if err != nil {
			return err
		}

		user, err := pocketid.NewUser(ctx, "myUser", &pocketid.UserArgs{
			Username: pulumi.String("my-app-user"),
			// Pocket-ID's API requires email even though the provider
			// marks it optional - see the "Known limitations" section
			// this comment doubles as a pointer to.
			Email:        pulumi.String("my-app-user@example.com"),
			DisplayName:  pulumi.String("My Application User"),
			UserGroupIds: pulumi.StringArray{group.ID()},
		})
		if err != nil {
			return err
		}

		client, err := pocketid.NewOidcClient(ctx, "myClient", &pocketid.OidcClientArgs{
			Name: pulumi.String("My Application"),
			CallbackUrls: pulumi.StringArray{
				pulumi.String("https://app.example.com/callback"),
			},
		})
		if err != nil {
			return err
		}

		_, err = pocketid.NewCustomClaims(ctx, "myClaims", &pocketid.CustomClaimsArgs{
			OwnerType: pulumi.String("user-group"),
			OwnerId:   group.ID(),
			Claims: pocketid.CustomClaimArray{
				pocketid.CustomClaimArgs{
					Key:   pulumi.String("department"),
					Value: pulumi.String("engineering"),
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("userGroupId", group.ID())
		ctx.Export("username", user.Username)
		ctx.Export("clientId", client.ClientId)
		return nil
	})
}
