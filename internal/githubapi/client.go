// Package githubapi is a minimal hand-rolled GitHub GraphQL v4 client:
// no third-party GraphQL library, just POST + json.Decode.
package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const graphQLEndpoint = "https://api.github.com/graphql"

// Client issues authenticated GraphQL requests against the GitHub API.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a Client authenticated with a GitHub PAT.
func NewClient(token string) *Client {
	return &Client{token: token, httpClient: &http.Client{}}
}

type graphQLRequest struct {
	Query     string `json:"query"`
	Variables any    `json:"variables"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

// Do executes query with the given variables and decodes the "data" field into out.
func (c *Client) Do(ctx context.Context, query string, variables any, out any) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphQLEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github graphql: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("github graphql error: %s", gqlResp.Errors[0].Message)
	}

	if out != nil {
		if err := json.Unmarshal(gqlResp.Data, out); err != nil {
			return fmt.Errorf("decoding data: %w", err)
		}
	}
	return nil
}
