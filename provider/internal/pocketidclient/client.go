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

// Package pocketidclient is a typed client for the Pocket-ID REST API
// (https://pocket-id.org/docs/api/endpoints), covering the operations needed
// by the Pulumi provider. Each API domain lives in its own file.
package pocketidclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

// Client talks to a single Pocket-ID instance.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New creates a Client for the instance rooted at baseURL (e.g.
// "https://id.example.com"). A nil hc falls back to http.DefaultClient.
func New(baseURL, apiKey string, hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: hc}
}

// APIError is returned when Pocket-ID responds with a status >= 400.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("pocket-id api: %d: %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether err is an APIError with a 404 status.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// Do issues a request and JSON-decodes the response into out (when non-nil
// and the body is non-empty). body is JSON-encoded when non-nil. It is the
// building block of every typed method and an escape hatch for tests.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var reader io.Reader
	contentType := ""
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(b)
		contentType = "application/json"
	}
	data, err := c.send(ctx, method, path, query, reader, contentType, "application/json")
	if err != nil {
		return err
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// DoMultipart uploads data as the "file" field of a multipart/form-data body.
// The part carries filename and a content type detected from the data (see
// DetectContentType); any response body is discarded.
func (c *Client) DoMultipart(
	ctx context.Context, method, path string, query url.Values, filename string, data []byte,
) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", DetectContentType(data))
	part, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("building multipart body: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("building multipart body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("building multipart body: %w", err)
	}
	_, err = c.send(ctx, method, path, query, &buf, w.FormDataContentType(), "application/json")
	return err
}

// GetBytes issues a GET and returns the raw (non-JSON) response body, e.g. an image.
func (c *Client) GetBytes(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.send(ctx, http.MethodGet, path, query, nil, "", "*/*")
}

// DetectContentType sniffs the MIME type of an image. net/http cannot tell SVG
// from XML or plain text, so SVG documents are recognised separately.
func DetectContentType(data []byte) string {
	head := data
	if len(head) > 1024 {
		head = head[:1024]
	}
	if ct := http.DetectContentType(data); strings.HasPrefix(ct, "image/") {
		return ct
	}
	if bytes.Contains(head, []byte("<svg")) {
		return "image/svg+xml"
	}
	return http.DetectContentType(data)
}

// send performs a request, returns the response body and maps statuses >= 400 to *APIError.
func (c *Client) send(
	ctx context.Context, method, path string, query url.Values, body io.Reader, contentType, accept string,
) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", accept)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: apiMessage(data)}
	}
	return data, nil
}

// apiMessage extracts {"error": "..."} from an error body, else the truncated raw body.
func apiMessage(data []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &e) == nil && e.Error != "" {
		return e.Error
	}
	const maxLen = 500
	if s := string(data); len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return string(data)
}

// page is the envelope of every paginated list endpoint.
type page[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		CurrentPage int `json:"currentPage"`
		TotalPages  int `json:"totalPages"`
	} `json:"pagination"`
}

// listAll walks every page of a paginated endpoint (100 items per page) and
// returns the concatenated items. query may carry extra filters (e.g. search).
func listAll[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	var all []T
	for n := 1; ; n++ {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		q.Set("pagination[page]", strconv.Itoa(n))
		q.Set("pagination[limit]", "100")

		var p page[T]
		if err := c.Do(ctx, http.MethodGet, path, q, nil, &p); err != nil {
			return nil, err
		}
		all = append(all, p.Data...)
		if n >= p.Pagination.TotalPages || len(p.Data) == 0 {
			return all, nil
		}
	}
}
