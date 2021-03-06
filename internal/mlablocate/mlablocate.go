// Package mlablocate contains a locate.measurementlab.net client.
package mlablocate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/ooni/probe-engine/model"
)

// NewRequestFunc is the function to create a new request.
type NewRequestFunc func(ctx context.Context, URL *url.URL) (*http.Request, error)

// Client is a locate.measurementlab.net client.
type Client struct {
	HTTPClient *http.Client
	Hostname   string
	Logger     model.Logger
	NewRequest NewRequestFunc
	Scheme     string
	UserAgent  string
}

// NewClient creates a new locate.measurementlab.net client.
func NewClient(httpClient *http.Client, logger model.Logger, userAgent string) *Client {
	return &Client{
		HTTPClient: httpClient,
		Hostname:   "locate.measurementlab.net",
		Logger:     logger,
		NewRequest: NewRequestDefault(),
		Scheme:     "https",
		UserAgent:  userAgent,
	}
}

// NewRequestDefault return the default implementation of the c.NewRequest
// for creating a new HTTP request for locate.measurementlab.net.
func NewRequestDefault() NewRequestFunc {
	return func(ctx context.Context, URL *url.URL) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	}
}

// NewRequestWithProxy returns a new request factory that tells to the
// locate.measurementlab.net service that we're using a proxy such that
// the returned host is good for us, not for the proxy.
func NewRequestWithProxy(probeIP string) NewRequestFunc {
	return func(ctx context.Context, URL *url.URL) (*http.Request, error) {
		values := URL.Query()
		values.Set("ip", probeIP)
		URL.RawQuery = values.Encode()
		return http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	}
}

type locateResult struct {
	FQDN string `json:"fqdn"`
}

// Query performs a locate.measurementlab.net query.
func (c *Client) Query(ctx context.Context, tool string) (string, error) {
	URL := &url.URL{
		Scheme: c.Scheme,
		Host:   c.Hostname,
		Path:   tool,
	}
	req, err := c.NewRequest(ctx, URL)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", c.UserAgent)
	c.Logger.Debugf("mlablocate: GET %s", URL.String())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("mlablocate: non-200 status code: %d", resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	c.Logger.Debugf("mlablocate: %s", string(data))
	var result locateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if result.FQDN == "" {
		return "", errors.New("mlablocate: returned empty FQDN")
	}
	return result.FQDN, nil
}
