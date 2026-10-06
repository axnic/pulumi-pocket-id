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
	"sort"
)

// apiHits returns how many API-domain requests the fake has served.
func apiHits(f *fakePocketID) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.store("apiHits"))
}

func init() {
	fakeRegistrars = append(fakeRegistrars, func(f *fakePocketID) {
		hit := func() { f.store("apiHits")[f.genID("hit-")] = nil }

		// --- APIs ---
		f.handle("POST /api/apis", func(w http.ResponseWriter, r *http.Request) {
			hit()
			b := readBody(r)
			id := f.genID("api-")
			f.store("apis")[id] = map[string]any{
				"id": id, keyName: b[keyName], "resource": b["resource"],
				keyCreatedAt: "2025-01-01T00:00:00Z", "allowCimdClients": false, "permissions": []any{},
			}
			writeJSON(w, http.StatusCreated, f.store("apis")[id])
		})
		f.handle("GET /api/apis/{id}", func(w http.ResponseWriter, r *http.Request) {
			hit()
			a, ok := f.store("apis")[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			writeJSON(w, http.StatusOK, a)
		})
		f.handle("PUT /api/apis/{id}", func(w http.ResponseWriter, r *http.Request) {
			hit()
			a, ok := f.store("apis")[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			a[keyName] = readBody(r)[keyName]
			writeJSON(w, http.StatusOK, a)
		})
		f.handle("DELETE /api/apis/{id}", func(w http.ResponseWriter, r *http.Request) {
			hit()
			id := r.PathValue("id")
			if _, ok := f.store("apis")[id]; !ok {
				notFound(w)
				return
			}
			delete(f.store("apis"), id)
			w.WriteHeader(http.StatusNoContent)
		})
		f.handle("PUT /api/apis/{id}/permissions", func(w http.ResponseWriter, r *http.Request) {
			hit()
			a, ok := f.store("apis")[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			existing := map[string]string{}
			allowed := map[string]bool{}
			for _, p := range a["permissions"].([]any) {
				pm := p.(map[string]any)
				existing[pm[keyKey].(string)] = pm["id"].(string)
				allowed[pm[keyKey].(string)] = pm["allowedForCimdClients"] == true
			}
			perms := []any{}
			in, _ := readBody(r)["permissions"].([]any)
			for _, p := range in {
				pm := p.(map[string]any)
				key, _ := pm[keyKey].(string)
				id, ok := existing[key]
				if !ok {
					id = f.genID("perm-")
				}
				perms = append(perms, map[string]any{
					"id": id, keyKey: key, keyName: pm[keyName], keyDescription: pm[keyDescription],
					"allowedForCimdClients": allowed[key],
				})
			}
			a["permissions"] = perms
			writeJSON(w, http.StatusOK, a)
		})
		f.handle("PUT /api/apis/{id}/cimd-access", func(w http.ResponseWriter, r *http.Request) {
			hit()
			a, ok := f.store("apis")[r.PathValue("id")]
			if !ok {
				notFound(w)
				return
			}
			b := readBody(r)
			ids := map[string]bool{}
			for _, id := range b["permissionIds"].([]any) {
				ids[id.(string)] = true
			}
			for _, p := range a["permissions"].([]any) {
				pm := p.(map[string]any)
				pm["allowedForCimdClients"] = ids[pm["id"].(string)]
			}
			a["allowCimdClients"] = b["enabled"]
			w.WriteHeader(http.StatusNoContent)
		})

		// --- API client grants (stored by "<apiId>/<clientId>") ---
		f.handle("GET /api/apis/{id}/clients", func(w http.ResponseWriter, r *http.Request) {
			hit()
			apiID := r.PathValue("id")
			if _, ok := f.store("apis")[apiID]; !ok {
				notFound(w)
				return
			}
			var keys []string
			for k, g := range f.store("apiGrants") {
				if g["apiId"] == apiID {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			items := []any{}
			for _, k := range keys {
				g := f.store("apiGrants")[k]
				items = append(items, map[string]any{
					"client": map[string]any{"id": g[keyClientID], keyName: "client",
						"clientType": "confidential"},
					keyClientAccess:              g[keyClientAccess],
					"clientPermissionIds":        g["clientPermissionIds"],
					keyUserDelegatedAccess:       g[keyUserDelegatedAccess],
					"userDelegatedPermissionIds": g["userDelegatedPermissionIds"],
					"cimdGrantedAccess":          false,
					"cimdGrantedPermissionIds":   []any{},
				})
			}
			paginated(w, items)
		})
		f.handle("PUT /api/apis/{id}/clients/{clientId}", func(w http.ResponseWriter, r *http.Request) {
			hit()
			apiID, clientID := r.PathValue("id"), r.PathValue(keyClientID)
			if _, ok := f.store("apis")[apiID]; !ok {
				notFound(w)
				return
			}
			b := readBody(r)
			f.store("apiGrants")[apiID+"/"+clientID] = map[string]any{
				"apiId": apiID, keyClientID: clientID,
				keyClientAccess: b[keyClientAccess], "clientPermissionIds": b["clientPermissionIds"],
				keyUserDelegatedAccess: b[keyUserDelegatedAccess], "userDelegatedPermissionIds": b["userDelegatedPermissionIds"],
			}
			writeJSON(w, http.StatusOK, b)
		})
		f.handle("DELETE /api/apis/{id}/clients/{clientId}", func(w http.ResponseWriter, r *http.Request) {
			hit()
			k := r.PathValue("id") + "/" + r.PathValue(keyClientID)
			if _, ok := f.store("apiGrants")[k]; !ok {
				notFound(w)
				return
			}
			delete(f.store("apiGrants"), k)
			w.WriteHeader(http.StatusNoContent)
		})
	})
}
