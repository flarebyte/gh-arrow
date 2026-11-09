// Package github implements read-only GitHub.com access using gh credentials.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
)

const RequestTimeout = 30 * time.Second
const MaxJSONBytes = 64 << 20

type Client struct {
	HTTP  *http.Client
	token string
}
type httpStatus int

func (s httpStatus) Error() string { return fmt.Sprintf("HTTP %d", s) }

// New obtains credentials without a shell or an interactive authentication flow.
// No credentials is valid for public repositories. HTTP errors guide login when needed.
func New(ctx context.Context) (*Client, error) {
	authCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(authCtx, "gh", "auth", "token", "--hostname", "github.com")
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "GH_DEBUG=")
	data, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if errors.Is(err, exec.ErrNotFound) {
		return nil, diagnostic.New("ARROW_AUTH", "GitHub CLI is required; install gh and run gh auth login separately")
	}
	if authCtx.Err() != nil {
		return nil, diagnostic.New("ARROW_AUTH", "credential lookup timed out; check gh authentication separately")
	}
	token := ""
	if err == nil {
		token = strings.TrimSpace(string(data))
	}
	return &Client{HTTP: &http.Client{Timeout: RequestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, token: token}, nil
}
func (c *Client) get(ctx context.Context, endpoint, accept string) (*http.Response, error) {
	// All callers supply fixed API paths; API links and redirect URLs are never followed.
	if !strings.HasPrefix(endpoint, "/") || strings.HasPrefix(endpoint, "//") {
		return nil, diagnostic.New("ARROW_NETWORK", "invalid API endpoint")
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+endpoint, nil)
		if err != nil {
			return nil, diagnostic.New("ARROW_USAGE", "invalid API path or ref")
		}
		req.Header.Set("Accept", accept)
		req.Header.Set("User-Agent", "gh-arrow")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		response, err := c.HTTP.Do(req)
		if err == nil && response.StatusCode == http.StatusOK {
			return response, nil
		}
		if ctx.Err() != nil {
			if response != nil {
				response.Body.Close()
			}
			return nil, ctx.Err()
		}
		retry := err != nil
		code := "ARROW_NETWORK"
		message := "GitHub API request failed"
		if response != nil {
			status := response.StatusCode
			// Inspect rate-limit messages only for classification; never echo remote bodies.
			body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
			response.Body.Close()
			switch {
			case status == 401:
				code = "ARROW_AUTH"
				message = "GitHub authentication failed; run gh auth login separately"
			case status == 429 || status == 403 && (response.Header.Get("X-RateLimit-Remaining") == "0" || response.Header.Get("Retry-After") != "" || strings.Contains(strings.ToLower(string(body)), "rate limit")):
				code = "ARROW_RATE_LIMIT"
				message = "GitHub API rate limit reached; retry later"
			case status == 403:
				code = "ARROW_PERMISSION"
				message = "GitHub API access denied"
			case status == 404 || status == 409:
				code = "ARROW_NOT_FOUND"
				message = "repository, ref, or object not found or inaccessible"
				if c.token == "" {
					message += "; for private repositories, run gh auth login separately"
				}
			case status >= 500:
				retry = true
			}
			message = fmt.Sprintf("%s (HTTP %d)", message, status)
		}
		if !retry || attempt == 2 {
			var cause error
			if response != nil {
				cause = httpStatus(response.StatusCode)
			}
			return nil, &diagnostic.Error{Code: code, Message: message, Cause: cause}
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	panic("unreachable")
}
func (c *Client) json(ctx context.Context, endpoint string, value any) error {
	response, err := c.get(ctx, endpoint, "application/vnd.github+json")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxJSONBytes+1))
	if err != nil {
		return diagnostic.New("ARROW_NETWORK", "failed reading GitHub API response")
	}
	if len(data) > MaxJSONBytes {
		return diagnostic.New("ARROW_INCOMPLETE", "GitHub metadata exceeds 64 MiB response limit")
	}
	if err := json.Unmarshal(data, value); err != nil {
		return diagnostic.New("ARROW_NETWORK", "invalid GitHub API response")
	}
	return nil
}
