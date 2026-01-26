package moodle

import (
  "encoding/json"
  "fmt"
  "regexp"
  "strings"
)

type Course struct {
  ID        int    `json:"id"`
  Fullname  string `json:"fullname"`
  Shortname string `json:"shortname"`
  Category  string `json:"category"`
  ViewURL   string `json:"viewUrl"`
}

type moodleAPICourse struct {
  ID             int    `json:"id"`
  Fullname       string `json:"fullname"`
  Shortname      string `json:"shortname"`
  CourseCategory string `json:"coursecategory"`
  ViewURL        string `json:"viewurl"`
}

type moodleAPIData struct {
  Courses []moodleAPICourse `json:"courses"`
}

type moodleAPIResponse struct {
  Error     bool           `json:"error"`
  Exception string         `json:"exception"`
  Data      *moodleAPIData `json:"data"`
}

type moodleAPIRequest struct {
  Index      int                    `json:"index"`
  MethodName string                 `json:"methodname"`
  Args       map[string]interface{} `json:"args"`
}

func (c *Client) FetchCourses() ([]Course, error) {
  sesskey, err := c.GetSesskey()
  if err != nil {
    return nil, err
  }

  apiURL := fmt.Sprintf("%s/lib/ajax/service.php?sesskey=%s&info=core_course_get_enrolled_courses_by_timeline_classification", strings.TrimRight(c.BaseURL, "/"), sesskey)

  payload := []moodleAPIRequest{
    {
      Index:      0,
      MethodName: "core_course_get_enrolled_courses_by_timeline_classification",
      Args: map[string]interface{}{
        "offset":           0,
        "limit":            0,
        "classification":   "all",
        "sort":             "fullname",
        "customfieldname":  "",
        "customfieldvalue": "",
        "requiredfields": []string{
          "id",
          "fullname",
          "shortname",
          "showcoursecategory",
          "showshortname",
          "visible",
          "enddate",
        },
      },
    },
  }

  resp, err := c.PostJSON(apiURL, payload, nil)
  if err != nil {
    return nil, err
  }
  if err := ensureOK(resp, 2048); err != nil {
    return nil, err
  }

  var response []moodleAPIResponse
  if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
    return nil, err
  }
  if len(response) == 0 {
    return nil, fmt.Errorf("empty api response")
  }

  result := response[0]
  if result.Error || result.Data == nil {
    return nil, fmt.Errorf("moodle api error: %s", result.Exception)
  }

  filtered := result.Data.Courses
  if c.School.CategoryFilter != nil {
    filtered = make([]moodleAPICourse, 0, len(result.Data.Courses))
    for _, course := range result.Data.Courses {
      if ShouldIncludeCategory(course.CourseCategory, c.School) {
        filtered = append(filtered, course)
      }
    }
  }

  courses := make([]Course, 0, len(filtered))
  for _, course := range filtered {
    courses = append(courses, Course{
      ID:        course.ID,
      Fullname:  cleanCourseName(course.Fullname, c.School.CourseNamePatterns),
      Shortname: course.Shortname,
      Category:  course.CourseCategory,
      ViewURL:   course.ViewURL,
    })
  }

  return courses, nil
}

func (c *Client) GetSesskey() (string, error) {
  if c.sesskey != "" {
    return c.sesskey, nil
  }
  html, err := c.FetchPage("/my/")
  if err != nil {
    return "", err
  }

  rePrimary := regexp.MustCompile(`"sesskey":"([^"]+)"`)
  match := rePrimary.FindStringSubmatch(html)
  if len(match) > 1 {
    c.sesskey = match[1]
    return c.sesskey, nil
  }

  reFallback := regexp.MustCompile(`sesskey=([a-zA-Z0-9]+)`) // fallback
  match = reFallback.FindStringSubmatch(html)
  if len(match) > 1 {
    c.sesskey = match[1]
    return c.sesskey, nil
  }

  return "", fmt.Errorf("could not extract sesskey from Moodle page")
}

func cleanCourseName(name string, patterns []*regexp.Regexp) string {
  cleaned := name
  for _, pattern := range patterns {
    cleaned = pattern.ReplaceAllString(cleaned, "")
  }
  cleaned = strings.TrimSpace(cleaned)
  if cleaned == "" {
    return name
  }
  return cleaned
}
