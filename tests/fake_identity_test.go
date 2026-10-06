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
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Identity domain of the fake Pocket-ID: users, user groups (with members),
// custom claims and signup tokens. DTO shapes follow the Pocket-ID swagger.
func init() {
	fakeRegistrars = append(fakeRegistrars, func(f *fakePocketID) {
		users, groups, tokens := f.store("users"), f.store("groups"), f.store("signup-tokens")

		userDTO := func(u map[string]any) map[string]any {
			out := map[string]any{}
			for k, v := range u {
				out[k] = v
			}
			gs := []any{}
			for _, g := range groups {
				for _, id := range g[keyUserIDs].([]string) {
					if id == u["id"] {
						gs = append(gs, groupMinimal(g))
					}
				}
			}
			out["userGroups"] = gs
			return out
		}
		groupDTO := func(g map[string]any) map[string]any {
			out := groupMinimal(g)
			us := []any{}
			for _, id := range g[keyUserIDs].([]string) {
				if u, ok := users[id]; ok {
					us = append(us, userDTO(u))
				}
			}
			out["users"] = us
			return out
		}
		match := func(r *http.Request, fields ...string) func(map[string]any) bool {
			s := strings.ToLower(r.URL.Query().Get("search"))
			return func(d map[string]any) bool {
				if s == "" {
					return true
				}
				for _, f := range fields {
					if v, _ := d[f].(string); strings.Contains(strings.ToLower(v), s) {
						return true
					}
				}
				return false
			}
		}

		// --- users ---
		f.handle("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			u := map[string]any{
				"id": f.genID("usr-"), keyUsername: body[keyUsername], keyEmail: body[keyEmail],
				"emailVerified": body["emailVerified"] == true, keyFirstName: strOr(body[keyFirstName]),
				keyLastName: body[keyLastName], keyDisplayName: strOr(body[keyDisplayName]),
				keyIsAdmin: body[keyIsAdmin] == true, keyLocale: body[keyLocale], keyDisabled: body[keyDisabled] == true,
				keyCustomClaims: []any{},
			}
			users[u["id"].(string)] = u
			writeJSON(w, http.StatusCreated, userDTO(u))
		})
		f.handle("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
			ok := match(r, keyUsername, keyEmail, keyFirstName, keyLastName)
			items := []any{}
			for _, u := range users {
				if ok(u) {
					items = append(items, userDTO(u))
				}
			}
			paginated(w, items)
		})
		f.handle("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			u, ok := users[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			writeJSON(w, http.StatusOK, userDTO(u))
		})
		f.handle("PUT /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			u, ok := users[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			body := readBody(r)
			for _, k := range []string{keyUsername, keyEmail, "emailVerified", keyFirstName, keyLastName,
				keyDisplayName, keyIsAdmin, keyLocale, keyDisabled} {
				u[k] = body[k]
			}
			writeJSON(w, http.StatusOK, userDTO(u))
		})
		f.handle("DELETE /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := users[r.PathValue("id")]; !ok {
				notFound(w)
				return
			}
			delete(users, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		})

		// --- user groups ---
		f.handle("POST /api/user-groups", func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			g := map[string]any{
				"id": f.genID("grp-"), keyName: body[keyName], keyFriendlyName: body[keyFriendlyName],
				keyCustomClaims: []any{}, keyUserIDs: []string{},
			}
			groups[g["id"].(string)] = g
			writeJSON(w, http.StatusCreated, groupDTO(g))
		})
		f.handle("GET /api/user-groups", func(w http.ResponseWriter, r *http.Request) {
			ok := match(r, keyName, keyFriendlyName)
			items := []any{}
			for _, g := range groups {
				if ok(g) {
					items = append(items, groupMinimal(g))
				}
			}
			paginated(w, items)
		})
		f.handle("GET /api/user-groups/{id}", func(w http.ResponseWriter, r *http.Request) {
			g, ok := groups[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			writeJSON(w, http.StatusOK, groupDTO(g))
		})
		f.handle("PUT /api/user-groups/{id}", func(w http.ResponseWriter, r *http.Request) {
			g, ok := groups[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			body := readBody(r)
			g[keyName], g[keyFriendlyName] = body[keyName], body[keyFriendlyName]
			writeJSON(w, http.StatusOK, groupDTO(g))
		})
		f.handle("DELETE /api/user-groups/{id}", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := groups[r.PathValue("id")]; !ok {
				notFound(w)
				return
			}
			delete(groups, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		})
		f.handle("PUT /api/user-groups/{id}/users", func(w http.ResponseWriter, r *http.Request) {
			g, ok := groups[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			ids := []string{}
			raw, _ := readBody(r)[keyUserIDs].([]any)
			for _, v := range raw {
				ids = append(ids, v.(string))
			}
			g[keyUserIDs] = ids
			writeJSON(w, http.StatusOK, groupDTO(g))
		})

		// --- custom claims (full replacement) ---
		setClaims := func(store map[string]map[string]any) func(http.ResponseWriter, *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				d, ok := store[r.PathValue("id")]
				if !ok {
					notFound(w)
					return
				}
				var claims []any
				if raw, ok := identityReadArray(r); ok {
					claims = raw
				}
				if claims == nil {
					claims = []any{}
				}
				d[keyCustomClaims] = claims
				writeJSON(w, http.StatusOK, claims)
			}
		}
		f.handle("PUT /api/custom-claims/user/{id}", setClaims(users))
		f.handle("PUT /api/custom-claims/user-group/{id}", setClaims(groups))

		// --- signup tokens ---
		f.handle("POST /api/signup-tokens", func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			ttl, _ := body[keyTTL].(float64)
			now := time.Now().UTC().Truncate(time.Second)
			gs := []any{}
			ids, _ := body[keyUserGroupIDs].([]any)
			for _, id := range ids {
				if g, ok := groups[id.(string)]; ok {
					gs = append(gs, groupMinimal(g))
				}
			}
			t := map[string]any{
				"id": f.genID("sgn-"), keyToken: f.genID("secret-"),
				keyCreatedAt: now.Format(time.RFC3339),
				keyExpiresAt: now.Add(time.Duration(ttl) * time.Second).Format(time.RFC3339),
				"usageCount": 0, keyUsageLimit: body[keyUsageLimit], "userGroups": gs,
			}
			tokens[t["id"].(string)] = t
			writeJSON(w, http.StatusCreated, t)
		})
		f.handle("GET /api/signup-tokens", func(w http.ResponseWriter, _ *http.Request) {
			items := []any{}
			for _, t := range tokens {
				items = append(items, t)
			}
			paginated(w, items)
		})
		f.handle("DELETE /api/signup-tokens/{id}", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := tokens[r.PathValue("id")]; !ok {
				notFound(w)
				return
			}
			delete(tokens, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func groupMinimal(g map[string]any) map[string]any {
	return map[string]any{
		"id": g["id"], keyName: g[keyName], keyFriendlyName: g[keyFriendlyName],
		keyCustomClaims: g[keyCustomClaims], "userCount": len(g[keyUserIDs].([]string)),
	}
}

func strOr(v any) string {
	s, _ := v.(string)
	return s
}

// identityReadArray decodes a JSON array request body.
func identityReadArray(r *http.Request) ([]any, bool) {
	var arr []any
	err := json.NewDecoder(r.Body).Decode(&arr)
	return arr, err == nil
}
