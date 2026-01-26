package moodle

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "strings"
)

func (c *Client) buildURL(path string) string {
  if strings.HasPrefix(path, "http") {
    return path
  }
  return strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
  req, err := http.NewRequest(method, c.buildURL(path), body)
  if err != nil {
    return nil, err
  }
  req.Header.Set("Cookie", c.Cookies)
  req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
  return req, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
  return c.http.Do(req)
}

func (c *Client) Get(path string) (*http.Response, error) {
  req, err := c.newRequest(http.MethodGet, path, nil)
  if err != nil {
    return nil, err
  }
  return c.Do(req)
}

func (c *Client) PostJSON(path string, payload any, headers map[string]string) (*http.Response, error) {
  data, err := json.Marshal(payload)
  if err != nil {
    return nil, err
  }
  req, err := c.newRequest(http.MethodPost, path, bytes.NewReader(data))
  if err != nil {
    return nil, err
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("X-Requested-With", "XMLHttpRequest")
  for k, v := range headers {
    req.Header.Set(k, v)
  }
  return c.Do(req)
}

func readResponseBody(resp *http.Response, limit int64) (string, error) {
  if resp.Body == nil {
    return "", nil
  }
  defer resp.Body.Close()
  data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
  if err != nil {
    return "", err
  }
  return string(data), nil
}

func ensureOK(resp *http.Response, limit int64) error {
  if resp.StatusCode >= 200 && resp.StatusCode < 300 {
    return nil
  }
  body, _ := readResponseBody(resp, limit)
  return fmt.Errorf("request failed: %s (%s)", resp.Status, strings.TrimSpace(body))
}
