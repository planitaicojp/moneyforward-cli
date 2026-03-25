package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	cerrors "github.com/planitaicojp/moneyforward-cli/internal/errors"
)

const defaultTimeout = 30 * time.Second

// Client is an HTTP client for the Money Forward Cloud API.
type Client struct {
	http      *http.Client
	token     string
	userAgent string
	verbose   bool
}

// New creates a new unauthenticated API client.
func New(version string, verbose bool) *Client {
	return &Client{
		http:      &http.Client{Timeout: defaultTimeout},
		userAgent: "mf-cli/" + version,
		verbose:   verbose,
	}
}

// NewWithToken creates an API client pre-configured with a Bearer token.
func NewWithToken(token, version string, verbose bool) *Client {
	c := New(version, verbose)
	c.token = token
	return c
}

// Do executes an HTTP request with retry logic for 429 and 5xx responses.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	var (
		resp    *http.Response
		err     error
		maxTry  = 3
		backoff = time.Second
	)

	for attempt := 0; attempt < maxTry; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		// Clone the request body for retry if needed.
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, err = io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("reading request body: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		if c.verbose {
			fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL)
		}

		resp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}

		if c.verbose {
			fmt.Fprintf(os.Stderr, "< HTTP %d\n", resp.StatusCode)
		}

		// Retry on 429 (Too Many Requests).
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
					time.Sleep(time.Duration(secs) * time.Second)
				}
			}
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			continue
		}

		// Retry on 5xx server errors.
		if resp.StatusCode >= 500 && attempt < maxTry-1 {
			resp.Body.Close()
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			continue
		}

		return resp, nil
	}

	return resp, err
}

// DoJSON sends a JSON request and unmarshals the JSON response into target.
// If target is nil, the response body is discarded.
func (c *Client) DoJSON(method, url string, body interface{}, target interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if c.verbose && len(respBytes) > 0 {
		fmt.Fprintf(os.Stderr, "< body: %s\n", string(respBytes))
	}

	if resp.StatusCode >= 400 {
		// Try to parse a JSON error body.
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
			Code    string `json:"code"`
		}
		_ = json.Unmarshal(respBytes, &apiErr)
		msg := apiErr.Message
		if msg == "" {
			msg = apiErr.Error
		}
		if msg == "" {
			msg = string(respBytes)
		}
		return &cerrors.APIError{
			StatusCode: resp.StatusCode,
			Code:       apiErr.Code,
			Message:    msg,
		}
	}

	if target != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, target); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}
