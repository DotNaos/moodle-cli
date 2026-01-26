package moodle

import (
  "regexp"
)

func (c *Client) FetchCourseResources(courseID string) ([]Resource, string, error) {
  html, err := c.FetchPage("/course/view.php?id=" + courseID)
  if err != nil {
    return nil, "", err
  }

  resources := ParseResources(html, courseID, c.BaseURL)

  contextID := ""
  reContext := regexp.MustCompile(`"contextid"\s*:\s*(\d+)`)
  if m := reContext.FindStringSubmatch(html); len(m) > 1 {
    contextID = m[1]
  } else {
    reFallback := regexp.MustCompile(`downloadcontent\.php\?contextid=(\d+)`)
    if m := reFallback.FindStringSubmatch(html); len(m) > 1 {
      contextID = m[1]
    }
  }

  return resources, contextID, nil
}
