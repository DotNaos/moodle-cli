package cli

import (
	"errors"
	"fmt"
	"strings"

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

	nameCounts := make(map[string]int, len(courses))
	for _, course := range courses {
		key := strings.ToLower(strings.TrimSpace(course.Fullname))
		nameCounts[key]++
	}

	out := make([]string, 0, len(courses))
	for _, course := range courses {
		value := course.Fullname
		if strings.TrimSpace(value) == "" {
			value = fmt.Sprintf("Course %d", course.ID)
		}
		if nameCounts[strings.ToLower(strings.TrimSpace(value))] > 1 {
			value = fmt.Sprintf("%s [id:%d]", value, course.ID)
		}

		desc := fmt.Sprintf("id:%d", course.ID)
		if course.Shortname != "" && !strings.EqualFold(course.Shortname, course.Fullname) {
			desc = fmt.Sprintf("id:%d short:%s", course.ID, course.Shortname)
		}

		out = append(out, formatCompValue(value, desc))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeCourseOrResourceIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeCourseIDs(cmd, args, toComplete)
	}

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

	courseID, err := resolveCourseIDFromCourses(courses, args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	resources, _, err := client.FetchCourseResources(courseID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	nameCounts := make(map[string]int, len(resources))
	for _, res := range resources {
		key := strings.ToLower(strings.TrimSpace(res.Name))
		nameCounts[key]++
	}

	out := make([]string, 0, len(resources))
	for _, res := range resources {
		value := res.Name
		if strings.TrimSpace(value) == "" {
			value = fmt.Sprintf("Resource %s", res.ID)
		}
		if nameCounts[strings.ToLower(strings.TrimSpace(value))] > 1 {
			value = fmt.Sprintf("%s [id:%s]", value, res.ID)
		}

		desc := fmt.Sprintf("id:%s", res.ID)
		if res.SectionName != "" {
			desc = fmt.Sprintf("id:%s section:%s", res.ID, res.SectionName)
		}
		if res.Type != "" {
			desc = fmt.Sprintf("%s type:%s", desc, res.Type)
		}

		out = append(out, formatCompValue(value, desc))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func formatCompValue(value string, desc string) string {
	if desc == "" {
		return value
	}
	return value + "\t" + desc
}
