package cli

import (
  "errors"
  "fmt"

  "github.com/DotNaos/moodle-cli/internal/moodle"
  "github.com/spf13/cobra"
)

func completeCourseIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
  session, err := moodle.LoadSession(opts.SessionPath)
  if err != nil {
    return nil, cobra.ShellCompDirectiveNoFileComp
  }
  client, err := moodle.NewClient(session)
  if err != nil {
    return nil, cobra.ShellCompDirectiveNoFileComp
  }
  if err := client.ValidateSession(); err != nil {
    if errors.Is(err, moodle.ErrSessionExpired) {
      return nil, cobra.ShellCompDirectiveNoFileComp
    }
    return nil, cobra.ShellCompDirectiveNoFileComp
  }

  courses, err := client.FetchCourses()
  if err != nil {
    return nil, cobra.ShellCompDirectiveNoFileComp
  }

  out := make([]string, 0, len(courses))
  for _, course := range courses {
    out = append(out, formatCompValue(fmt.Sprintf("%d", course.ID), course.Fullname))
  }
  return out, cobra.ShellCompDirectiveNoFileComp
}

func completeCourseOrResourceIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
  if len(args) == 0 {
    return completeCourseIDs(cmd, args, toComplete)
  }

  courseID := args[0]
  session, err := moodle.LoadSession(opts.SessionPath)
  if err != nil {
    return nil, cobra.ShellCompDirectiveNoFileComp
  }
  client, err := moodle.NewClient(session)
  if err != nil {
    return nil, cobra.ShellCompDirectiveNoFileComp
  }
  if err := client.ValidateSession(); err != nil {
    if errors.Is(err, moodle.ErrSessionExpired) {
      return nil, cobra.ShellCompDirectiveNoFileComp
    }
    return nil, cobra.ShellCompDirectiveNoFileComp
  }

  resources, _, err := client.FetchCourseResources(courseID)
  if err != nil {
    return nil, cobra.ShellCompDirectiveNoFileComp
  }

  out := make([]string, 0, len(resources))
  for _, res := range resources {
    out = append(out, formatCompValue(res.ID, res.Name))
  }
  return out, cobra.ShellCompDirectiveNoFileComp
}

func formatCompValue(value string, desc string) string {
  if desc == "" {
    return value
  }
  return value + "\t" + desc
}
