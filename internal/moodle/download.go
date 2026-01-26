package moodle

import (
  "bytes"
  "fmt"
  "io"
)

type DownloadResult struct {
  Data        []byte
  ContentType string
}

func (c *Client) DownloadFileToBuffer(url string) (DownloadResult, error) {
  resp, err := c.Get(url)
  if err != nil {
    return DownloadResult{}, err
  }
  if err := ensureOK(resp, 2048); err != nil {
    return DownloadResult{}, err
  }
  data, err := io.ReadAll(resp.Body)
  if err != nil {
    return DownloadResult{}, err
  }
  contentType := resp.Header.Get("Content-Type")
  if contentType == "" {
    contentType = "application/octet-stream"
  }
  return DownloadResult{Data: data, ContentType: contentType}, nil
}

func (c *Client) DownloadFileToString(url string) (string, error) {
  result, err := c.DownloadFileToBuffer(url)
  if err != nil {
    return "", err
  }
  if len(result.Data) == 0 {
    return "", fmt.Errorf("empty response")
  }
  return string(bytes.TrimSpace(result.Data)), nil
}
