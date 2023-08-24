package common

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/spyzhov/ajson"
)

// ErrorHandler allows the caller to inject their own HTTP error handling logic.
// All non-2xx responses will be passed to the error handler. If the error handler
// returns nil, then the error is ignored and the caller is responsible for handling
// the error. If the error handler returns an error, then that error is returned
// to the caller, as-is. Both the response as well as the response body are passed
// to the error handler as arguments.
type ErrorHandler func(rsp *http.Response, body []byte) error

// JSONHTTPClient is an HTTP client which makes certain assumptions, such as
// that the response body is JSON. It also handles OAuth access token refreshes.
type JSONHTTPClient struct {
	Client       AuthenticatedHTTPClient
	ErrorHandler ErrorHandler
}

// Get makes a GET request to the given URL and returns the response body as a JSON object.
// If the response is not a 2xx, an error is returned. If the response is a 401, the caller should
// refresh the access token and retry the request. If errorHandler is nil, then the default error
// handler is used. If not, the caller can inject their own error handling logic.
func (j *JSONHTTPClient) Get(ctx context.Context,
	url string, headers ...Header,
) (*ajson.Node, error) {
	// Make the request, get the response body
	res, body, err := j.httpGet(ctx, url, headers) //nolint:bodyclose
	if err != nil {
		return nil, err
	}

	// Ensure the response is JSON
	ct := res.Header.Get("Content-Type")
	if len(ct) > 0 {
		mimeType, _, err := mime.ParseMediaType(ct)
		if err != nil {
			return nil, fmt.Errorf("failed to parse content type: %w", err)
		}

		if mimeType != "application/json" {
			return nil, fmt.Errorf("%w: expected content type to be application/json, got %s", ErrNotJSON, mimeType)
		}
	}

	// Unmarshall the response body into JSON
	jsonBody, err := ajson.Unmarshal(body)
	if err != nil {
		return nil, NewHTTPStatusError(res.StatusCode, fmt.Errorf("failed to unmarshall response body into JSON: %w", err))
	}

	return jsonBody, nil
}

func (j *JSONHTTPClient) httpGet(ctx context.Context, url string,
	headers []Header,
) (*http.Response, []byte, error) {
	req, err := makeJSONRequest(ctx, url, headers)
	if err != nil {
		return nil, nil, err
	}

	// Send the request
	res, err := j.Client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error sending request: %w", err)
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)

	defer func() {
		if res != nil && res.Body != nil {
			if closeErr := res.Body.Close(); closeErr != nil {
				slog.Warn("unable to close response body", "error", closeErr)
			}
		}
	}()

	if err != nil {
		return nil, nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Check the response status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		if j.ErrorHandler != nil {
			return nil, nil, j.ErrorHandler(res, body)
		}

		return nil, nil, InterpretError(res, body)
	}

	return res, body, nil
}

func makeJSONRequest(ctx context.Context, url string, headers []Header) (*http.Request, error) {
	// Create a new GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Request JSON
	req.Header.Add("Accept", "application/json")

	// Apply any custom headers
	for _, hdr := range headers {
		req.Header.Add(hdr.Key, hdr.Value)
	}

	return req, nil
}
