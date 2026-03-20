package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var downloadAll bool
var downloadOutputDir string

var downloadCmd = &cobra.Command{
	Use:               "download file <course-id|name|current|0> <resource-id|name|current|0>",
	Short:             "Download a file from a course",
	Long:              "Download one or more files from a course to your filesystem.\n\nUse --all to download all files in the course. The course and file can be specified by ID, name, `current`, `0`, or a positive index.",
	Example:           "  moodle download file 12345 67890\n  moodle download file current current\n  moodle download file 0 0\n  moodle download file 12345 --all -o ./downloads",
	ValidArgsFunction: completeDownloadFile,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("expected 'file' and course")
		}
		if args[0] != "file" {
			return fmt.Errorf("expected 'file' subcommand")
		}
		if downloadAll {
			if len(args) != 2 {
				return fmt.Errorf("expected only course when using --all")
			}
			return nil
		}
		if len(args) != 3 {
			return fmt.Errorf("expected course and file")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		courseID, err := resolveCourseIDWithOptions(client, args[1], selectorOptions{})
		if err != nil {
			return err
		}
		resources, _, err := client.FetchCourseResources(courseID)
		if err != nil {
			return err
		}

		if downloadAll {
			return downloadAllResources(client, resources, downloadOutputDir)
		}

		target, err := resolveResourceWithOptions(client, courseID, resources, args[2], selectorOptions{})
		if err != nil {
			return err
		}
		if target.Type != "resource" {
			return fmt.Errorf("resource %s is not a file", target.ID)
		}
		return downloadResourceToPath(client, *target, downloadOutputDir)
	},
}

func init() {
	downloadCmd.Flags().BoolVar(&downloadAll, "all", false, "Download all files in the course")
	downloadCmd.Flags().StringVarP(&downloadOutputDir, "output-dir", "o", "", "Output directory (or file path for single download)")
}

func downloadAllResources(client *moodle.Client, resources []moodle.Resource, outputPath string) error {
	outputPath = resolveDefaultOutputDir(outputPath)
	if err := ensureDir(outputPath); err != nil {
		return err
	}

	for _, res := range resources {
		if res.Type != "resource" {
			continue
		}
		path, err := resolveOutputPath(outputPath, res)
		if err != nil {
			return err
		}
		if err := downloadResourceToFile(client, res, path); err != nil {
			return err
		}
	}
	return nil
}

func downloadResourceToPath(client *moodle.Client, res moodle.Resource, outputPath string) error {
	path, err := resolveOutputPath(outputPath, res)
	if err != nil {
		return err
	}
	return downloadResourceToFile(client, res, path)
}

func downloadResourceToFile(client *moodle.Client, res moodle.Resource, path string) error {
	result, err := client.DownloadFileToBuffer(res.URL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, result.Data, 0o644)
}

func resolveOutputPath(outputPath string, res moodle.Resource) (string, error) {
	outputPath = resolveDefaultOutputDir(outputPath)
	if info, err := os.Stat(outputPath); err == nil {
		if info.IsDir() {
			return filepath.Join(outputPath, buildResourceFilename(res)), nil
		}
		return outputPath, nil
	}

	if strings.HasSuffix(outputPath, string(os.PathSeparator)) {
		if err := ensureDir(outputPath); err != nil {
			return "", err
		}
		return filepath.Join(outputPath, buildResourceFilename(res)), nil
	}

	if filepath.Ext(outputPath) == "" {
		if err := ensureDir(outputPath); err != nil {
			return "", err
		}
		return filepath.Join(outputPath, buildResourceFilename(res)), nil
	}

	return outputPath, nil
}

func resolveDefaultOutputDir(outputPath string) string {
	if outputPath == "" {
		return opts.ExportDir
	}
	return outputPath
}

func buildResourceFilename(res moodle.Resource) string {
	name := strings.TrimSpace(res.Name)
	if name == "" {
		name = "resource-" + res.ID
	}
	name = sanitizeFilename(name)
	if filepath.Ext(name) == "" && res.FileType != "" {
		name += "." + res.FileType
	}
	return name
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func sanitizeFilename(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return replacer.Replace(value)
}
