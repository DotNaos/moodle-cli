package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var currentLectureResultForCompletion = func(client *moodle.Client) (currentLectureResult, error) {
	return resolveCurrentLectureResult(client, selectorOptions{})
}

func completeCourseIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	out := []string{
		formatCompValue("current", "Current lecture course"),
		formatCompValue("0", "Current lecture course"),
	}

	session, err := moodle.LoadSession(opts.SessionPath)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	client, err := moodle.NewClient(session)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	if err := client.ValidateSession(); err != nil {
		if errors.Is(err, moodle.ErrSessionExpired) {
			return out, cobra.ShellCompDirectiveNoFileComp
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}

	courses, err := client.FetchCourses()
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	updateCurrentCourseCompletionDescriptions(out, client)

	nameCounts := make(map[string]int, len(courses))
	for _, course := range courses {
		key := strings.ToLower(strings.TrimSpace(course.Fullname))
		nameCounts[key]++
	}

	for index, course := range courses {
		out = append(out, formatCompValue(fmt.Sprintf("%d", index+1), course.Fullname))
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

func completeResourcesForCourseArg(courseArg string) ([]string, cobra.ShellCompDirective) {
	out := []string{
		formatCompValue("current", "Current or top-ranked material"),
		formatCompValue("0", "Current or top-ranked material"),
	}

	session, err := moodle.LoadSession(opts.SessionPath)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	client, err := moodle.NewClient(session)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	if err := client.ValidateSession(); err != nil {
		if errors.Is(err, moodle.ErrSessionExpired) {
			return out, cobra.ShellCompDirectiveNoFileComp
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}

	courses, err := client.FetchCourses()
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}

	currentResult, _ := currentLectureResultForCompletion(client)
	currentCourse := currentResult.Course
	courseID, err := resolveCourseIDFromCoursesWithCurrent(courses, courseArg, currentCourse)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}

	resources, _, err := client.FetchCourseResources(courseID)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	updateCurrentResourceCompletionDescriptions(out, courseID, currentResult)

	nameCounts := make(map[string]int, len(resources))
	for _, res := range resources {
		key := strings.ToLower(strings.TrimSpace(res.Name))
		nameCounts[key]++
	}

	indexedResources := orderedResourcesForCompletion(courseID, resources, currentResult)
	for index, res := range indexedResources {
		out = append(out, formatCompValue(fmt.Sprintf("%d", index+1), res.Name))
	}
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

func completeDownloadFile(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return []string{formatCompValue("file", "Download a file")}, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 && args[0] == "file" {
		return completeCourseIDs(nil, nil, "")
	}
	if len(args) == 2 && args[0] == "file" {
		return completeResourcesForCourseArg(args[1])
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completePrintCourseFile(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		results, directive := completeCourseIDs(nil, nil, "")
		prefixed := []string{
			formatCompValue("current-course", "Current lecture and current material"),
			formatCompValue("current-resource", "Current lecture and current material"),
			formatCompValue("current-ressource", "Current lecture and current material"),
		}
		return append(prefixed, results...), directive
	}
	if len(args) == 1 {
		return completeResourcesForCourseArg(args[0])
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeListSelectionArgs(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		results, directive := completeCourseIDs(nil, nil, "")
		prefixed := []string{
			formatCompValue("courses", "List courses"),
			formatCompValue("files", "List files in a course"),
			formatCompValue("timetable", "List timetable entries"),
			formatCompValue("current-course", "Current lecture and current material"),
			formatCompValue("current-resource", "Current lecture and current material"),
			formatCompValue("current-ressource", "Current lecture and current material"),
		}
		return append(prefixed, results...), directive
	}
	if len(args) == 1 {
		return completeResourcesForCourseArg(args[0])
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeOpenDirectArgs(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		results, directive := completeCourseIDs(nil, nil, "")
		prefixed := []string{
			formatCompValue("course", "Open a course"),
			formatCompValue("resource", "Open a resource"),
			formatCompValue("current-resource", "Open the current lecture material"),
			formatCompValue("current-course", "Open the current lecture material"),
			formatCompValue("current-ressource", "Open the current lecture material"),
		}
		return append(prefixed, results...), directive
	}
	if len(args) == 1 {
		switch args[0] {
		case "course", "resource", "current-lecture":
			return nil, cobra.ShellCompDirectiveNoFileComp
		default:
			return completeResourcesForCourseArg(args[0])
		}
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeOpenResourceArgs(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeCourseIDs(nil, nil, "")
	}
	if len(args) == 1 {
		return completeResourcesForCourseArg(args[0])
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeOpenTargets(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{
		formatCompValue("course", "Open a course"),
		formatCompValue("resource", "Open a resource"),
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeExportCourse(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return []string{formatCompValue("course", "Export a course")}, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 && args[0] == "course" {
		return completeCourseIDs(nil, nil, "")
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeSchoolIDs(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	out := make([]string, 0, len(moodle.Schools))
	for _, s := range moodle.Schools {
		out = append(out, formatCompValue(s.ID, s.Name))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func updateCurrentCourseCompletionDescriptions(out []string, client *moodle.Client) {
	result, err := currentLectureResultForCompletion(client)
	if err != nil || result.Event == nil || result.Course == nil {
		return
	}
	desc := fmt.Sprintf("%s -> %s", result.Event.Summary, result.Course.Title)
	out[0] = formatCompValue("current", desc)
	out[1] = formatCompValue("0", desc)
}

func updateCurrentResourceCompletionDescriptions(out []string, courseID string, result currentLectureResult) {
	if result.Course == nil || result.Material == nil {
		return
	}
	if fmt.Sprintf("%d", result.Course.ID) != strings.TrimSpace(courseID) {
		return
	}
	desc := fmt.Sprintf("%s (current material)", result.Material.Label)
	out[0] = formatCompValue("current", desc)
	out[1] = formatCompValue("0", desc)
}

func orderedResourcesForCompletion(courseID string, resources []moodle.Resource, result currentLectureResult) []moodle.Resource {
	if result.Course == nil {
		return fileResources(resources)
	}
	if fmt.Sprintf("%d", result.Course.ID) != strings.TrimSpace(courseID) {
		return fileResources(resources)
	}
	return orderedFileResources(resources, resourceIDs(result.Resources))
}
