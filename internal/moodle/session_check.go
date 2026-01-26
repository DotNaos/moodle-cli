package moodle

import (
  "errors"
  "io"
  "net/http"
  "strings"
  "time"
)

func (c *Client) ValidateSession() error {
  // Use a client that does not follow redirects so we can detect login redirects.
  redirectClient := &http.Client{
    Timeout: 30 * time.Second,
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
      return http.ErrUseLastResponse
    },
  }

  req, err := c.newRequest(http.MethodGet, "/my/courses.php", nil)
  if err != nil {
    return err
  }

  resp, err := redirectClient.Do(req)
  if err != nil {
    return err
  }
  defer resp.Body.Close()

  if resp.StatusCode >= 300 && resp.StatusCode < 400 {
    location := resp.Header.Get("Location")
    if strings.Contains(strings.ToLower(location), "login") {
      return ErrSessionExpired
    }
  }

  if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    return errors.New(resp.Status)
  }

  body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
  snippet := strings.ToLower(string(body))
  if strings.Contains(snippet, "login") && strings.Contains(snippet, "username") {
    return ErrSessionExpired
  }

  return nil
}
