package moodle

import (
  "errors"
  "fmt"
  "regexp"
  "strings"
  "time"

  "github.com/playwright-community/playwright-go"
)

type LoginOptions struct {
  SchoolID string
  Username string
  Password string
  Headless bool
  Timeout  time.Duration
}

type LoginResult struct {
  Cookies  string
  SchoolID string
}

func resolveSchool(id string) (SchoolConfig, error) {
  if id == "" {
    return GetDefaultSchool(), nil
  }
  school := GetSchool(id)
  if school == nil {
    return SchoolConfig{}, fmt.Errorf("unknown school id: %s", id)
  }
  return *school, nil
}

func LoginWithPlaywright(options LoginOptions) (LoginResult, error) {
  school, err := resolveSchool(options.SchoolID)
  if err != nil {
    return LoginResult{}, err
  }
  timeout := options.Timeout
  if timeout == 0 {
    timeout = 120 * time.Second
  }

  if options.Headless && (options.Username == "" || options.Password == "") {
    return LoginResult{}, errors.New("headless login requires --username and --password")
  }

  pw, err := playwright.Run()
  if err != nil {
    return LoginResult{}, err
  }
  defer pw.Stop()

  browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(options.Headless)})
  if err != nil {
    return LoginResult{}, err
  }
  defer browser.Close()

  context, err := browser.NewContext()
  if err != nil {
    return LoginResult{}, err
  }

  page, err := context.NewPage()
  if err != nil {
    return LoginResult{}, err
  }

  _, err = page.Goto(school.LoginURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
  if err != nil {
    return LoginResult{}, err
  }

  if options.Username != "" && options.Password != "" {
    if err := fillLoginForm(page, school, options.Username, options.Password, timeout); err != nil {
      return LoginResult{}, err
    }
  }

  cookies, err := waitForSessionCookie(context, school, timeout)
  if err != nil {
    return LoginResult{}, err
  }

  return LoginResult{Cookies: cookies, SchoolID: school.ID}, nil
}

func fillLoginForm(page playwright.Page, school SchoolConfig, username string, password string, timeout time.Duration) error {
  continueRegex := `(?i)continue|weiter|next|fortsetzen|proceed`
  continueRe := regexp.MustCompile(continueRegex)

  clickContinueIfNeeded := func() (bool, error) {
    roles := []playwright.AriaRole{*playwright.AriaRoleButton, *playwright.AriaRoleLink}
    for _, role := range roles {
      locator := page.GetByRole(role, playwright.PageGetByRoleOptions{Name: continueRe}).First()
      count, err := locator.Count()
      if err != nil {
        continue
      }
      if count > 0 {
        visible, _ := locator.IsVisible()
        if visible {
          if err := locator.Click(); err != nil {
            return false, err
          }
          return true, nil
        }
      }
    }

    inputs := page.Locator(`input[type="submit"], input[type="button"]`)
    count, err := inputs.Count()
    if err != nil {
      return false, nil
    }
    for i := 0; i < count; i++ {
      input := inputs.Nth(i)
      value, _ := input.GetAttribute("value")
      if value != "" && continueRe.MatchString(value) {
        visible, _ := input.IsVisible()
        if visible {
          if err := input.Click(); err != nil {
            return false, err
          }
          return true, nil
        }
      }
    }

    return false, nil
  }

  usernameField := page.Locator(school.Selectors.Username).First()
  count, _ := usernameField.Count()
  if count > 0 {
    visible, _ := usernameField.IsVisible()
    if !visible {
      clicked, err := clickContinueIfNeeded()
      if err != nil {
        return err
      }
      if clicked {
        _, _ = page.WaitForSelector(school.Selectors.Username, playwright.PageWaitForSelectorOptions{
          State:   playwright.WaitForSelectorStateVisible,
          Timeout: playwright.Float(float64(minDuration(timeout, 10*time.Second).Milliseconds())),
        })
      }
    }
    if err := usernameField.Fill(username); err != nil {
      return err
    }
  }

  passwordField := page.Locator(school.Selectors.Password).First()
  hasPasswordField := false
  if count, _ := passwordField.Count(); count > 0 {
    hasPasswordField = true
    if err := passwordField.Fill(password); err != nil {
      return err
    }
  }

  submitButton := page.Locator(school.Selectors.Submit).First()
  if count, _ := submitButton.Count(); count > 0 {
    if err := submitButton.Click(); err != nil {
      return err
    }
  }

  passwordVisible := false
  if _, err := page.WaitForSelector(school.Selectors.Password, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(5000)}); err == nil {
    passwordVisible = true
  }

  if passwordVisible && !hasPasswordField {
    updatedPassword := page.Locator(school.Selectors.Password).First()
    if err := updatedPassword.Fill(password); err != nil {
      return err
    }
    updatedSubmit := page.Locator(school.Selectors.Submit).First()
    if count, _ := updatedSubmit.Count(); count > 0 {
      if err := updatedSubmit.Click(); err != nil {
        return err
      }
    }
  }

  return page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
    State:   playwright.LoadStateDomcontentloaded,
    Timeout: playwright.Float(float64(timeout.Milliseconds())),
  })
}

func waitForSessionCookie(context playwright.BrowserContext, school SchoolConfig, timeout time.Duration) (string, error) {
  end := time.Now().Add(timeout)
  for time.Now().Before(end) {
    cookies, err := context.Cookies(school.MoodleURL)
    if err != nil {
      return "", err
    }
    hasSession := false
    parts := make([]string, 0, len(cookies))
    for _, cookie := range cookies {
      nameLower := strings.ToLower(cookie.Name)
      if strings.Contains(nameLower, "moodlesession") && !strings.Contains(nameLower, "test") {
        hasSession = true
      }
      parts = append(parts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
    }
    if hasSession {
      return strings.Join(parts, "; "), nil
    }
    time.Sleep(1 * time.Second)
  }
  return "", errors.New("login timed out before session cookies were set")
}

func minDuration(a, b time.Duration) time.Duration {
  if a < b {
    return a
  }
  return b
}
