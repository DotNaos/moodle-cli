package moodle

import (
  "encoding/json"
  "fmt"
  "time"
)

type Deadline struct {
  ID        int       `json:"id"`
  Name      string    `json:"name"`
  TimeStart time.Time `json:"timeStart"`
  TimeSort  time.Time `json:"timeSort"`
  CourseID  int       `json:"courseId"`
  Course    string    `json:"course"`
  URL       string    `json:"url"`
  Type      string    `json:"type"`
}

type calendarEvent struct {
  ID        int    `json:"id"`
  Name      string `json:"name"`
  TimeStart int64  `json:"timestart"`
  TimeSort  int64  `json:"timesort"`
  URL       string `json:"url"`
  EventType string `json:"eventtype"`
  Course    *struct {
    ID       int    `json:"id"`
    Fullname string `json:"fullname"`
    Short    string `json:"shortname"`
  } `json:"course"`
}

type calendarEventsData struct {
  Events []calendarEvent `json:"events"`
}

type calendarEventsResponse struct {
  Error     bool                `json:"error"`
  Exception interface{}         `json:"exception"`
  Data      *calendarEventsData  `json:"data"`
}

func (c *Client) FetchDeadlines(from time.Time, to time.Time) ([]Deadline, error) {
  sesskey, err := c.GetSesskey()
  if err != nil {
    return nil, err
  }

  apiURL := fmt.Sprintf("%s/lib/ajax/service.php?sesskey=%s&info=core_calendar_get_action_events_by_timesort", c.BaseURL, sesskey)

  payload := []moodleAPIRequest{
    {
      Index:      0,
      MethodName: "core_calendar_get_action_events_by_timesort",
      Args: map[string]interface{}{
        "limitnum": 200,
        "timesortfrom": from.Unix(),
        "timesortto":   to.Unix(),
        "limittononsuspendedevents": true,
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

  var response []calendarEventsResponse
  if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
    return nil, err
  }
  if len(response) == 0 {
    return nil, fmt.Errorf("empty api response")
  }

  result := response[0]
  if result.Error || result.Data == nil {
    return nil, fmt.Errorf("moodle api error: %v", result.Exception)
  }

  out := make([]Deadline, 0, len(result.Data.Events))
  for _, ev := range result.Data.Events {
    courseID := 0
    courseName := ""
    if ev.Course != nil {
      courseID = ev.Course.ID
      if ev.Course.Fullname != "" {
        courseName = ev.Course.Fullname
      } else {
        courseName = ev.Course.Short
      }
    }
    out = append(out, Deadline{
      ID:        ev.ID,
      Name:      ev.Name,
      TimeStart: time.Unix(ev.TimeStart, 0),
      TimeSort:  time.Unix(ev.TimeSort, 0),
      CourseID:  courseID,
      Course:    courseName,
      URL:       ev.URL,
      Type:      ev.EventType,
    })
  }

  return out, nil
}
