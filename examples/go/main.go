package main

import (
	pocketid "github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// A group of users and an OIDC client only that group may sign in to, plus an
// API the client is granted access to. Needs `pocket-id:baseUrl` and
// `pocket-id:apiKey` in the stack config (see README.md).
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		group, err := pocketid.NewUserGroup(ctx, "engineers", &pocketid.UserGroupArgs{
			Name:         pulumi.String("engineers"),
			FriendlyName: pulumi.String("Engineers"),
		})
		if err != nil {
			return err
		}

		user, err := pocketid.NewUser(ctx, "alice", &pocketid.UserArgs{
			Username:    pulumi.String("alice"),
			Email:       pulumi.String("alice@example.com"),
			FirstName:   pulumi.String("Alice"),
			LastName:    pulumi.String("Liddell"),
			DisplayName: pulumi.String("Alice Liddell"),
		})
		if err != nil {
			return err
		}

		// Authoritative: the group contains exactly these users.
		_, err = pocketid.NewUserGroupMembers(ctx, "engineersMembers", &pocketid.UserGroupMembersArgs{
			GroupId: group.ID(),
			UserIds: pulumi.StringArray{user.ID()},
		})
		if err != nil {
			return err
		}

		client, err := pocketid.NewOidcClient(ctx, "app", &pocketid.OidcClientArgs{
			Name:               pulumi.String("My Application"),
			CallbackUrls:       pulumi.StringArray{pulumi.String("https://app.example.com/callback")},
			LogoutCallbackUrls: pulumi.StringArray{pulumi.String("https://app.example.com/logout")},
			// Only members of these groups can sign in to the client.
			AllowedUserGroupIds: pulumi.StringArray{group.ID()},
		})
		if err != nil {
			return err
		}

		secret, err := pocketid.NewOidcClientSecret(ctx, "appSecret", &pocketid.OidcClientSecretArgs{
			ClientId: client.ID(),
		})
		if err != nil {
			return err
		}

		api, err := pocketid.NewApi(ctx, "backend", &pocketid.ApiArgs{
			Name:     pulumi.String("Backend API"),
			Resource: pulumi.String("https://api.example.com"),
			Permissions: pocketid.ApiPermissionArgsArray{
				pocketid.ApiPermissionArgsArgs{
					Key:         pulumi.String("read"),
					Name:        pulumi.String("Read"),
					Description: pulumi.String("Read access to the backend"),
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = pocketid.NewApiClientGrant(ctx, "appBackendAccess", &pocketid.ApiClientGrantArgs{
			ApiId:                api.ID(),
			ClientId:             client.ID(),
			ClientAccess:         pulumi.Bool(true),
			ClientPermissionKeys: pulumi.StringArray{pulumi.String("read")},
		})
		if err != nil {
			return err
		}

		ctx.Export("groupId", group.ID())
		ctx.Export("username", user.Username)
		ctx.Export("clientId", client.ClientId)
		ctx.Export("clientSecret", secret.Secret)
		ctx.Export("apiId", api.ID())
		return nil
	})
}
