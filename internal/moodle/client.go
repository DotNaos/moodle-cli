package moodle

import (
  "fmt"
  "net/http"
  "time"
)

type Client struct {
  BaseURL string
  Cookies string
  School  SchoolConfig
  http    *http.Client
  sesskey string
}

func NewClient(session Session) (*Client, error) {
  school, err := resolveSchool(session.SchoolID)
  if err != nil {
    return nil, err
  }
  if session.Cookies == "" {
    return nil, fmt.Errorf("session missing cookies")
  }
  return &Client{
    BaseURL: school.MoodleURL,
    Cookies: session.Cookies,
    School:  school,
    http: &http.Client{
      Timeout: 60 * time.Second,
    },
  }, nil
}

func (c *Client) FetchPage(path string) (string, error) {
  resp, err := c.Get(path)
  if err != nil {
    return "", err
  }
  if err := ensureOK(resp, 1024); err != nil {
    return "", err
  }
  return readResponseBody(resp, 10*1024*1024)
}
