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
package tests

import "net/http"

// appConfigRequired lists the keys the real PUT /api/application-configuration
// rejects the request without (dto.AppConfigUpdateDto "required").
var appConfigRequired = []string{
	"allowOwnAccountEdit", keyAllowUserSignups, keyAppName, "autoCreateOidcClientSecret", keyDisableAnimations,
	"emailApiKeyExpirationEnabled", "emailLoginNotificationEnabled", "emailOneTimeAccessAsAdminEnabled",
	"emailOneTimeAccessAsUnauthenticatedEnabled", "emailVerificationEnabled", "emailsVerified", "homePageUrl",
	"ldapEnabled", "ldapSkipCertVerify", "ldapSoftDeleteUsers", "requireUserEmail", keySessionDuration,
	"smtpSkipCertVerify", "smtpTls", "webauthnAllowSyncedPasskeys", "webauthnAuthenticatorAttachment",
	"webauthnUserVerification",
}

func init() {
	fakeRegistrars = append(fakeRegistrars, func(f *fakePocketID) {
		cfg := f.store("appconfig")
		// Pocket-ID defaults, all as strings. The "all" key holds the single row.
		cfg["all"] = map[string]any{
			keyAppName: "Pocket ID", keySessionDuration: "60", "homePageUrl": "/settings/account",
			"allowOwnAccountEdit": keyTrue, keyAllowUserSignups: keyDisabled, "autoCreateOidcClientSecret": keyFalse,
			keyDisableAnimations: keyFalse, "emailApiKeyExpirationEnabled": keyFalse, "emailLoginNotificationEnabled": keyFalse,
			"emailOneTimeAccessAsAdminEnabled": keyFalse, "emailOneTimeAccessAsUnauthenticatedEnabled": keyFalse,
			"emailVerificationEnabled": keyFalse, "emailsVerified": keyFalse, "ldapEnabled": keyFalse,
			"ldapSkipCertVerify": keyFalse, "ldapSoftDeleteUsers": keyTrue, "requireUserEmail": keyTrue,
			"smtpSkipCertVerify": keyFalse, "smtpTls": "none", "smtpPort": "587", "smtpHost": "", "smtpPassword": "",
			"webauthnAllowSyncedPasskeys": keyTrue, "webauthnAuthenticatorAttachment": "any",
			"webauthnUserVerification": "preferred", "accentColor": "default",
			// Not part of the update DTO: must survive a PUT untouched.
			"instanceId": "fake-instance",
		}

		f.handle("GET /api/application-configuration/all", func(w http.ResponseWriter, _ *http.Request) {
			out := []any{}
			for k, v := range cfg["all"] {
				out = append(out, map[string]any{keyKey: k, "type": "string", "value": v, keyIsPublic: false})
			}
			writeJSON(w, http.StatusOK, out)
		})

		f.handle("PUT /api/application-configuration", func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			for _, k := range appConfigRequired {
				if _, ok := body[k]; !ok {
					writeJSON(w, http.StatusBadRequest, map[string]string{keyError: "missing required field " + k})
					return
				}
			}
			for k, v := range body {
				if _, ok := v.(string); !ok {
					writeJSON(w, http.StatusBadRequest, map[string]string{keyError: "field " + k + " must be a string"})
					return
				}
				cfg["all"][k] = v
			}
			writeJSON(w, http.StatusOK, []any{})
		})

		f.handle("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{ //nolint:gosec // not a credential
				"issuer":                 "https://id.example.com",
				"authorization_endpoint": "https://id.example.com/authorize",
				"token_endpoint":         "https://id.example.com/api/oidc/token",
				"userinfo_endpoint":      "https://id.example.com/api/oidc/userinfo",
				"jwks_uri":               "https://id.example.com/.well-known/jwks.json",
				"scopes_supported":       []string{"openid", "profile", keyEmail},
			})
		})
	})
}
