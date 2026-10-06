// Copyright 2025, axnic.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package provider

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// appConfigID is the fixed ID of the ApplicationConfiguration singleton.
const appConfigID = "app-config"

// ApplicationConfiguration manages the instance-wide settings of Pocket-ID
// (general, email, SMTP, LDAP and WebAuthn). It is a singleton: an instance
// always has exactly one configuration, so creating the resource only applies
// settings and deleting it leaves the configuration as it is.
//
// Only the fields set in the program are managed. Pocket-ID stores every
// value as a string and its update endpoint needs the complete set of
// mandatory keys, so each update reads the current configuration, overlays the
// managed fields and writes everything back.
type ApplicationConfiguration struct{}

// ApplicationConfigurationArgs are the inputs of the ApplicationConfiguration
// resource. The pulumi tag of each field is the Pocket-ID configuration key.
type ApplicationConfigurationArgs struct {
	// General.
	AppName                    *string `pulumi:"appName,optional"`
	SessionDuration            *int    `pulumi:"sessionDuration,optional"`
	HomePageURL                *string `pulumi:"homePageUrl,optional"`
	AccentColor                *string `pulumi:"accentColor,optional"`
	DisableAnimations          *bool   `pulumi:"disableAnimations,optional"`
	AllowOwnAccountEdit        *bool   `pulumi:"allowOwnAccountEdit,optional"`
	AllowUserSignups           *string `pulumi:"allowUserSignups,optional"`
	RequireUserEmail           *bool   `pulumi:"requireUserEmail,optional"`
	AutoCreateOidcClientSecret *bool   `pulumi:"autoCreateOidcClientSecret,optional"`
	EmailsVerified             *bool   `pulumi:"emailsVerified,optional"`
	CimdURLAllowlist           *string `pulumi:"cimdUrlAllowlist,optional"`
	SignupDefaultCustomClaims  *string `pulumi:"signupDefaultCustomClaims,optional"`
	SignupDefaultUserGroupIDs  *string `pulumi:"signupDefaultUserGroupIDs,optional"`

	// Email notifications.
	EmailAPIKeyExpirationEnabled               *bool `pulumi:"emailApiKeyExpirationEnabled,optional"`
	EmailLoginNotificationEnabled              *bool `pulumi:"emailLoginNotificationEnabled,optional"`
	EmailOneTimeAccessAsAdminEnabled           *bool `pulumi:"emailOneTimeAccessAsAdminEnabled,optional"`
	EmailOneTimeAccessAsUnauthenticatedEnabled *bool `pulumi:"emailOneTimeAccessAsUnauthenticatedEnabled,optional"`
	EmailVerificationEnabled                   *bool `pulumi:"emailVerificationEnabled,optional"`

	// SMTP.
	SMTPHost           *string `pulumi:"smtpHost,optional"`
	SMTPPort           *int    `pulumi:"smtpPort,optional"`
	SMTPFrom           *string `pulumi:"smtpFrom,optional"`
	SMTPUser           *string `pulumi:"smtpUser,optional"`
	SMTPPassword       *string `pulumi:"smtpPassword,optional" provider:"secret"`
	SMTPTLS            *string `pulumi:"smtpTls,optional"`
	SMTPSkipCertVerify *bool   `pulumi:"smtpSkipCertVerify,optional"`

	// LDAP.
	LDAPEnabled                        *bool   `pulumi:"ldapEnabled,optional"`
	LDAPURL                            *string `pulumi:"ldapUrl,optional"`
	LDAPBindDN                         *string `pulumi:"ldapBindDn,optional"`
	LDAPBindPassword                   *string `pulumi:"ldapBindPassword,optional" provider:"secret"`
	LDAPBase                           *string `pulumi:"ldapBase,optional"`
	LDAPUserSearchFilter               *string `pulumi:"ldapUserSearchFilter,optional"`
	LDAPUserGroupSearchFilter          *string `pulumi:"ldapUserGroupSearchFilter,optional"`
	LDAPSkipCertVerify                 *bool   `pulumi:"ldapSkipCertVerify,optional"`
	LDAPSoftDeleteUsers                *bool   `pulumi:"ldapSoftDeleteUsers,optional"`
	LDAPAdminGroupName                 *string `pulumi:"ldapAdminGroupName,optional"`
	LDAPAttributeUserUniqueIdentifier  *string `pulumi:"ldapAttributeUserUniqueIdentifier,optional"`
	LDAPAttributeUserUsername          *string `pulumi:"ldapAttributeUserUsername,optional"`
	LDAPAttributeUserEmail             *string `pulumi:"ldapAttributeUserEmail,optional"`
	LDAPAttributeUserFirstName         *string `pulumi:"ldapAttributeUserFirstName,optional"`
	LDAPAttributeUserLastName          *string `pulumi:"ldapAttributeUserLastName,optional"`
	LDAPAttributeUserDisplayName       *string `pulumi:"ldapAttributeUserDisplayName,optional"`
	LDAPAttributeUserProfilePicture    *string `pulumi:"ldapAttributeUserProfilePicture,optional"`
	LDAPAttributeGroupUniqueIdentifier *string `pulumi:"ldapAttributeGroupUniqueIdentifier,optional"`
	LDAPAttributeGroupName             *string `pulumi:"ldapAttributeGroupName,optional"`
	LDAPAttributeGroupMember           *string `pulumi:"ldapAttributeGroupMember,optional"`

	// WebAuthn.
	WebauthnAllowSyncedPasskeys     *bool   `pulumi:"webauthnAllowSyncedPasskeys,optional"`
	WebauthnAuthenticatorAttachment *string `pulumi:"webauthnAuthenticatorAttachment,optional"`
	WebauthnUserVerification        *string `pulumi:"webauthnUserVerification,optional"`
}

// ApplicationConfigurationState is what is persisted for the resource: the
// managed settings, with the same secrecy rules as the inputs.
type ApplicationConfigurationState struct {
	ApplicationConfigurationArgs
}

var _ infer.Annotated = (*ApplicationConfiguration)(nil)

// Annotate describes the resource.
func (r *ApplicationConfiguration) Annotate(a infer.Annotator) {
	a.Describe(r, "The instance-wide configuration of Pocket-ID. "+
		"It is a singleton with the fixed ID \"app-config\": only the fields that are set are managed, "+
		"the others keep their current value on the server, and destroying the resource leaves the "+
		"configuration untouched. Use at most one per instance.")
}

var _ infer.Annotated = (*ApplicationConfigurationArgs)(nil)

// Annotate provides schema descriptions for the configuration fields.
func (c *ApplicationConfigurationArgs) Annotate(a infer.Annotator) {
	a.Describe(&c.AppName, "The application name shown in the UI and in emails (1 to 30 characters).")
	a.Describe(&c.SessionDuration, "The session lifetime, in minutes.")
	a.Describe(&c.HomePageURL, "The path of the page users land on after signing in, e.g. \"/settings/account\".")
	a.Describe(&c.AccentColor, "The UI accent color, as \"default\" or a CSS color value.")
	a.Describe(&c.DisableAnimations, "Whether to disable the UI animations.")
	a.Describe(&c.AllowOwnAccountEdit, "Whether users may edit their own account details.")
	a.Describe(&c.AllowUserSignups, "Who may sign up by themselves: \"disabled\", \"withToken\" or \"open\".")
	a.Describe(&c.RequireUserEmail, "Whether an email address is required for every user.")
	a.Describe(&c.AutoCreateOidcClientSecret, "Whether a secret is generated automatically when an OIDC client is "+
		"created.")
	a.Describe(&c.EmailsVerified, "Whether emails are considered verified by default.")
	a.Describe(&c.CimdURLAllowlist, "The allowlist of URLs accepted as OIDC client metadata documents (CIMD), in the "+
		"format "+
		"Pocket-ID expects.")
	a.Describe(&c.SignupDefaultCustomClaims, "The custom claims given to new users, as the JSON array string Pocket-ID "+
		"expects.")
	a.Describe(&c.SignupDefaultUserGroupIDs, "The IDs of the user groups new users join, as the JSON array string "+
		"Pocket-ID expects.")

	a.Describe(&c.EmailAPIKeyExpirationEnabled, "Whether an email is sent when an API key is about to expire.")
	a.Describe(&c.EmailLoginNotificationEnabled, "Whether an email is sent when a user signs in from a new device.")
	a.Describe(&c.EmailOneTimeAccessAsAdminEnabled, "Whether administrators may send one-time access links by email.")
	a.Describe(&c.EmailOneTimeAccessAsUnauthenticatedEnabled, "Whether unauthenticated users may request a one-time "+
		"access link by email.")
	a.Describe(&c.EmailVerificationEnabled, "Whether users must verify their email address.")

	a.Describe(&c.SMTPHost, "The SMTP server host name.")
	a.Describe(&c.SMTPPort, "The SMTP server port.")
	a.Describe(&c.SMTPFrom, "The sender address of outgoing emails.")
	a.Describe(&c.SMTPUser, "The SMTP user name.")
	a.Describe(&c.SMTPPassword, "The SMTP password. It is a secret and is never read back from Pocket-ID: "+
		"the value in state is the one from the program, and it is empty after an import.")
	a.Describe(&c.SMTPTLS, "The SMTP transport security: \"none\", \"starttls\" or \"tls\".")
	a.Describe(&c.SMTPSkipCertVerify, "Whether to skip the verification of the SMTP server certificate.")

	a.Describe(&c.LDAPEnabled, "Whether LDAP synchronization is enabled.")
	a.Describe(&c.LDAPURL, "The LDAP server URL, e.g. \"ldaps://ldap.example.com:636\".")
	a.Describe(&c.LDAPBindDN, "The distinguished name used to bind to the LDAP server.")
	a.Describe(&c.LDAPBindPassword, "The LDAP bind password. It is a secret and is never read back from Pocket-ID: "+
		"the value in state is the one from the program, and it is empty after an import.")
	a.Describe(&c.LDAPBase, "The LDAP base DN under which users and groups are searched.")
	a.Describe(&c.LDAPUserSearchFilter, "The LDAP filter selecting users, e.g. \"(objectClass=person)\".")
	a.Describe(&c.LDAPUserGroupSearchFilter, "The LDAP filter selecting groups, e.g. \"(objectClass=groupOfNames)\".")
	a.Describe(&c.LDAPSkipCertVerify, "Whether to skip the verification of the LDAP server certificate.")
	a.Describe(&c.LDAPSoftDeleteUsers, "Whether users removed from LDAP are disabled instead of deleted.")
	a.Describe(&c.LDAPAdminGroupName, "The name of the LDAP group whose members become Pocket-ID administrators.")
	a.Describe(&c.LDAPAttributeUserUniqueIdentifier, "The LDAP attribute holding the unique identifier of a user.")
	a.Describe(&c.LDAPAttributeUserUsername, "The LDAP attribute mapped to the user name.")
	a.Describe(&c.LDAPAttributeUserEmail, "The LDAP attribute mapped to the email address.")
	a.Describe(&c.LDAPAttributeUserFirstName, "The LDAP attribute mapped to the first name.")
	a.Describe(&c.LDAPAttributeUserLastName, "The LDAP attribute mapped to the last name.")
	a.Describe(&c.LDAPAttributeUserDisplayName, "The LDAP attribute mapped to the display name.")
	a.Describe(&c.LDAPAttributeUserProfilePicture, "The LDAP attribute mapped to the profile picture.")
	a.Describe(&c.LDAPAttributeGroupUniqueIdentifier, "The LDAP attribute holding the unique identifier of a group.")
	a.Describe(&c.LDAPAttributeGroupName, "The LDAP attribute mapped to the group name.")
	a.Describe(&c.LDAPAttributeGroupMember, "The LDAP attribute listing the members of a group.")

	a.Describe(&c.WebauthnAllowSyncedPasskeys, "Whether passkeys synced between devices are accepted.")
	a.Describe(&c.WebauthnAuthenticatorAttachment, "The accepted authenticator attachment: \"any\", \"platform\" or "+
		"\"cross-platform\".")
	a.Describe(&c.WebauthnUserVerification, "The user verification requirement: \"required\" or \"preferred\".")
}

func (ApplicationConfiguration) Create(
	ctx context.Context, req infer.CreateRequest[ApplicationConfigurationArgs],
) (infer.CreateResponse[ApplicationConfigurationState], error) {
	state := ApplicationConfigurationState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[ApplicationConfigurationState]{ID: appConfigID, Output: state}, nil
	}
	if err := applyAppConfig(ctx, req.Inputs); err != nil {
		return infer.CreateResponse[ApplicationConfigurationState]{}, err
	}
	return infer.CreateResponse[ApplicationConfigurationState]{ID: appConfigID, Output: state}, nil
}

func (ApplicationConfiguration) Read(
	ctx context.Context, req infer.ReadRequest[ApplicationConfigurationArgs, ApplicationConfigurationState],
) (infer.ReadResponse[ApplicationConfigurationArgs, ApplicationConfigurationState], error) {
	type resp = infer.ReadResponse[ApplicationConfigurationArgs, ApplicationConfigurationState]
	values, err := clientFromContext(ctx).GetAppConfig(ctx)
	if err != nil {
		return resp{}, err
	}
	// Only the fields already managed are read back, otherwise every refresh
	// would show a diff for the settings the program does not set. After an
	// import (nothing managed yet) every field is read.
	tracked := req.Inputs
	if reflect.DeepEqual(tracked, ApplicationConfigurationArgs{}) {
		tracked = ApplicationConfigurationArgs{}
		forEachField(&tracked, func(_ string, _ bool, f reflect.Value) { allocField(f) })
	}
	var args ApplicationConfigurationArgs
	var firstErr error
	forEachField(&args, func(key string, secret bool, f reflect.Value) {
		src := fieldByKey(&tracked, key)
		if src.IsNil() {
			return
		}
		if secret {
			// Never returned meaningfully by the API: keep what the state holds.
			f.Set(fieldByKey(&req.State.ApplicationConfigurationArgs, key))
			return
		}
		if v, ok := values[key]; ok {
			if err := setFromString(f, v); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("configuration key %q: %w", key, err)
			}
		}
	})
	if firstErr != nil {
		return resp{}, firstErr
	}
	return resp{ID: appConfigID, Inputs: args, State: ApplicationConfigurationState{args}}, nil
}

func (ApplicationConfiguration) Update(
	ctx context.Context, req infer.UpdateRequest[ApplicationConfigurationArgs, ApplicationConfigurationState],
) (infer.UpdateResponse[ApplicationConfigurationState], error) {
	state := ApplicationConfigurationState{req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[ApplicationConfigurationState]{Output: state}, nil
	}
	if err := applyAppConfig(ctx, req.Inputs); err != nil {
		return infer.UpdateResponse[ApplicationConfigurationState]{}, err
	}
	return infer.UpdateResponse[ApplicationConfigurationState]{Output: state}, nil
}

// Delete does nothing: a Pocket-ID instance always has a configuration, so
// there is nothing to remove. The settings stay as they were last applied.
func (ApplicationConfiguration) Delete(
	_ context.Context, _ infer.DeleteRequest[ApplicationConfigurationState],
) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}

// applyAppConfig overlays the managed fields on the current configuration and
// writes the result back (the update endpoint needs the full set of keys).
func applyAppConfig(ctx context.Context, args ApplicationConfigurationArgs) error {
	client := clientFromContext(ctx)
	current, err := client.GetAppConfig(ctx)
	if err != nil {
		return err
	}
	body := map[string]string{}
	forEachField(&args, func(key string, _ bool, f reflect.Value) {
		if v, ok := current[key]; ok {
			body[key] = v
		}
		if !f.IsNil() {
			body[key] = formatValue(f.Elem())
		}
	})
	return client.UpdateAppConfig(ctx, body)
}

// forEachField calls fn with the configuration key, secrecy and (pointer)
// value of every field of args.
func forEachField(args *ApplicationConfigurationArgs, fn func(key string, secret bool, f reflect.Value)) {
	v := reflect.ValueOf(args).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		key, _, _ := strings.Cut(t.Field(i).Tag.Get("pulumi"), ",")
		fn(key, t.Field(i).Tag.Get("provider") == "secret", v.Field(i))
	}
}

func fieldByKey(args *ApplicationConfigurationArgs, key string) reflect.Value {
	var out reflect.Value
	forEachField(args, func(k string, _ bool, f reflect.Value) {
		if k == key {
			out = f
		}
	})
	return out
}

// allocField makes the pointer field non-nil (used to mark it as tracked).
func allocField(f reflect.Value) { f.Set(reflect.New(f.Type().Elem())) }

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int:
		return strconv.FormatInt(v.Int(), 10)
	default:
		return v.String()
	}
}

// setFromString parses the raw configuration value s into the pointer field f.
func setFromString(f reflect.Value, s string) error {
	p := reflect.New(f.Type().Elem())
	// The server reports unset booleans and integers as "": leave them nil.
	if s == "" && p.Elem().Kind() != reflect.String {
		return nil
	}
	switch p.Elem().Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		p.Elem().SetBool(b)
	case reflect.Int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		p.Elem().SetInt(int64(n))
	default:
		p.Elem().SetString(s)
	}
	f.Set(p)
	return nil
}

func init() { registerResource(infer.Resource(&ApplicationConfiguration{})) }
