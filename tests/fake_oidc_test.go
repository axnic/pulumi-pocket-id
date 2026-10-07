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

package tests

import (
	"net/http"
	"time"
)

// Stores used by the OIDC domain: "oidcClients", "oidcSecrets" (keyed by secret
// ID, with a keyClientID field) and "scimProviders" (keyed by provider ID).
func init() {
	fakeRegistrars = append(fakeRegistrars, func(f *fakePocketID) {
		clients := f.store("oidcClients")
		secrets := f.store("oidcSecrets")
		scim := f.store("scimProviders")

		clientDTO := func(c map[string]any) map[string]any {
			out := map[string]any{"pkceSupported": true, "allowedUserGroups": []any{}}
			for k, v := range c {
				out[k] = v
			}
			groups := []any{}
			for _, id := range c[keyAllowedUserGroupIDs].([]any) {
				groups = append(groups, map[string]any{"id": id, keyName: id, keyFriendlyName: id})
			}
			out["allowedUserGroups"] = groups
			delete(out, keyAllowedUserGroupIDs)
			// Like the real instance, the logo flags follow the stored images, not the request.
			_, out["hasLogo"] = f.store("images")["oidc/"+c["id"].(string)+"/light"]
			_, out["hasDarkLogo"] = f.store("images")["oidc/"+c["id"].(string)+"/dark"]
			out["clientType"] = "confidential"
			if c[keyIsPublic] == true {
				out["clientType"] = "public"
			}
			return out
		}
		secretDTO := func(s map[string]any, withValue bool) map[string]any {
			out := map[string]any{
				"id": s["id"], "prefix": s["prefix"], keyCreatedAt: s[keyCreatedAt], "isActive": true,
			}
			if s[keyExpiresAt] != nil {
				out[keyExpiresAt] = s[keyExpiresAt]
			}
			if withValue {
				out["secret"] = s["secret"]
			}
			return out
		}
		newSecret := func(clientID string, expiresAt any) map[string]any {
			id := f.genID("sec-")
			s := map[string]any{
				"id": id, keyClientID: clientID, "prefix": "pre" + id, "secret": "plain-" + id,
				keyCreatedAt: time.Now().UTC().Format(time.RFC3339), keyExpiresAt: expiresAt,
			}
			secrets[id] = s
			return s
		}
		// fill applies the writable client fields of a Create/Update DTO.
		fill := func(c, body map[string]any) {
			for _, k := range []string{
				keyName, keyDescription, "callbackURLs", "logoutCallbackURLs", "launchURL",
				keyIsPublic, "pkceEnabled", "requiresReauthentication", "requiresPushedAuthorizationRequests",
				"skipConsent", "isGroupRestricted", "accessTokenDurationMinutes", "refreshTokenDurationMinutes",
				"backchannelLogoutURL", "credentials",
			} {
				if v, ok := body[k]; ok {
					c[k] = v
				} else {
					delete(c, k)
				}
			}
			// Server defaults, like the real instance: no launch URL is null, no back-channel URL is "".
			if _, ok := c["launchURL"]; !ok {
				c["launchURL"] = nil
			}
			if f.legacyOIDC {
				delete(c, "backchannelLogoutURL")
			} else if _, ok := c["backchannelLogoutURL"]; !ok {
				c["backchannelLogoutURL"] = ""
			}
			if _, ok := c["accessTokenDurationMinutes"]; !ok {
				c["accessTokenDurationMinutes"] = 60
			}
		}

		// invalid mirrors the real validation: an optional URL sent as "" is a 400 (it must be omitted).
		invalid := func(w http.ResponseWriter, body map[string]any) bool {
			for k, msg := range map[string]string{
				"launchURL": "LaunchURL must be a valid URL", "backchannelLogoutURL": "BackchannelLogoutURL must be a valid URL",
			} {
				if v, ok := body[k].(string); ok && v == "" {
					writeJSON(w, http.StatusBadRequest, map[string]string{keyError: msg})
					return true
				}
			}
			return false
		}

		f.handle("POST /api/oidc/clients", func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if invalid(w, body) {
				return
			}
			id, _ := body["id"].(string)
			if id == "" {
				id = f.genID("oidc-")
			}
			if _, exists := clients[id]; exists {
				writeJSON(w, http.StatusBadRequest, map[string]string{keyError: "client id already in use"})
				return
			}
			c := map[string]any{"id": id, keyAllowedUserGroupIDs: []any{}}
			fill(c, body)
			clients[id] = c
			out := clientDTO(c)
			if c[keyIsPublic] != true {
				// Pocket-ID generates a first secret for confidential clients.
				out["createdSecret"] = secretDTO(newSecret(id, nil), true)
			}
			writeJSON(w, http.StatusCreated, out)
		})
		f.handle("GET /api/oidc/clients/{id}", func(w http.ResponseWriter, r *http.Request) {
			c, ok := clients[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			writeJSON(w, http.StatusOK, clientDTO(c))
		})
		f.handle("PUT /api/oidc/clients/{id}", func(w http.ResponseWriter, r *http.Request) {
			c, ok := clients[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			body := readBody(r)
			if invalid(w, body) {
				return
			}
			fill(c, body)
			writeJSON(w, http.StatusOK, clientDTO(c))
		})
		f.handle("DELETE /api/oidc/clients/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			if _, ok := clients[id]; !ok {
				notFound(w)
				return
			}
			delete(clients, id)
			for sid, s := range secrets {
				if s[keyClientID] == id {
					delete(secrets, sid)
				}
			}
			for sid, s := range scim {
				if s[keyOidcClientID] == id {
					delete(scim, sid)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		})
		f.handle("PUT /api/oidc/clients/{id}/allowed-user-groups", func(w http.ResponseWriter, r *http.Request) {
			c, ok := clients[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			ids, _ := readBody(r)[keyUserGroupIDs].([]any)
			if ids == nil {
				ids = []any{}
			}
			c[keyAllowedUserGroupIDs] = ids
			c["isGroupRestricted"] = len(ids) > 0
			writeJSON(w, http.StatusOK, clientDTO(c))
		})

		f.handle("GET /api/oidc/clients/{id}/secrets", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := clients[r.PathValue("id")]; !ok {
				notFound(w)
				return
			}
			out := []any{}
			for _, s := range secrets {
				if s[keyClientID] == r.PathValue("id") {
					out = append(out, secretDTO(s, false))
				}
			}
			writeJSON(w, http.StatusOK, out)
		})
		f.handle("POST /api/oidc/clients/{id}/secrets", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := clients[r.PathValue("id")]; !ok {
				notFound(w)
				return
			}
			writeJSON(w, http.StatusCreated, secretDTO(newSecret(r.PathValue("id"), readBody(r)[keyExpiresAt]), true))
		})
		f.handle("DELETE /api/oidc/clients/{id}/secrets/{secretId}", func(w http.ResponseWriter, r *http.Request) {
			s, ok := secrets[r.PathValue("secretId")]
			if !ok || s[keyClientID] != r.PathValue("id") {
				notFound(w)
				return
			}
			delete(secrets, r.PathValue("secretId"))
			w.WriteHeader(http.StatusNoContent)
		})

		scimDTO := func(s map[string]any) map[string]any {
			return map[string]any{
				"id": s["id"], keyEndpoint: s[keyEndpoint], keyCreatedAt: s[keyCreatedAt],
				"oidcClient": map[string]any{"id": s[keyOidcClientID]},
			}
		}
		f.handle("POST /api/scim/service-provider", func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			clientID, _ := body[keyOidcClientID].(string)
			if _, ok := clients[clientID]; !ok {
				notFound(w)
				return
			}
			s := map[string]any{
				"id": f.genID("scim-"), keyOidcClientID: clientID, keyEndpoint: body[keyEndpoint], keyToken: body[keyToken],
				keyCreatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			scim[s["id"].(string)] = s
			writeJSON(w, http.StatusCreated, scimDTO(s))
		})
		f.handle("PUT /api/scim/service-provider/{id}", func(w http.ResponseWriter, r *http.Request) {
			s, ok := scim[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			body := readBody(r)
			s[keyEndpoint], s[keyToken] = body[keyEndpoint], body[keyToken]
			writeJSON(w, http.StatusOK, scimDTO(s))
		})
		f.handle("DELETE /api/scim/service-provider/{id}", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := scim[r.PathValue("id")]; !ok {
				notFound(w)
				return
			}
			delete(scim, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		})
		f.handle("GET /api/oidc/clients/{id}/scim-service-provider", func(w http.ResponseWriter, r *http.Request) {
			for _, s := range scim {
				if s[keyOidcClientID] == r.PathValue("id") {
					writeJSON(w, http.StatusOK, scimDTO(s))
					return
				}
			}
			notFound(w)
		})
	})
}
