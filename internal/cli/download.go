package cli

import (
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var downloadAll bool
var downloadOutputDir string

type downloadedFileResult struct {
	ResourceID string `json:"resourceId" yaml:"resourceId"`
	Name       string `json:"name" yaml:"name"`
	Path       string `json:"path" yaml:"path"`
}

type downloadCommandResult struct {
	Action string                 `json:"action" yaml:"action"`
	Files  []downloadedFileResult `json:"files" yaml:"files"`
}

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
			result, err := downloadAllResources(client, resources, downloadOutputDir)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, result, func(w io.Writer) error {
				return nil
			})
		}

		target, err := resolveResourceWithOptions(client, courseID, resources, args[2], selectorOptions{})
		if err != nil {
			return err
		}
		if target.Type != "resource" {
			return fmt.Errorf("resource %s is not a file", target.ID)
		}
		path, err := downloadResourceToPath(client, *target, downloadOutputDir)
		if err != nil {
			return err
		}
		result := downloadCommandResult{
			Action: "download",
			Files: []downloadedFileResult{{
				ResourceID: target.ID,
				Name:       target.Name,
				Path:       path,
			}},
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return nil
		})
	},
}

func init() {
	downloadCmd.Flags().BoolVar(&downloadAll, "all", false, "Download all files in the course")
	downloadCmd.Flags().StringVarP(&downloadOutputDir, "output-dir", "o", "", "Output directory (or file path for single download)")
}

func downloadAllResources(client *moodle.Client, resources []moodle.Resource, outputPath string) (downloadCommandResult, error) {
	outputPath = resolveDefaultOutputDir(outputPath)
	if err := ensureDir(outputPath); err != nil {
		return downloadCommandResult{}, err
	}
	result := downloadCommandResult{
		Action: "download",
		Files:  []downloadedFileResult{},
	}

	for _, res := range resources {
		if res.Type != "resource" {
			continue
		}
		path, err := resolveOutputPath(outputPath, res)
		if err != nil {
			return downloadCommandResult{}, err
		}
		if err := downloadResourceToFile(client, res, path); err != nil {
			return downloadCommandResult{}, err
		}
		result.Files = append(result.Files, downloadedFileResult{
			ResourceID: res.ID,
			Name:       res.Name,
			Path:       path,
		})
	}
	return result, nil
}

func downloadResourceToPath(client *moodle.Client, res moodle.Resource, outputPath string) (string, error) {
	path, err := resolveOutputPath(outputPath, res)
	if err != nil {
		return "", err
	}
	return path, downloadResourceToFile(client, res, path)
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
	return buildDownloadedResourceFilename(res, "")
}

func buildDownloadedResourceFilename(res moodle.Resource, contentType string) string {
	name := strings.TrimSpace(res.Name)
	if name == "" {
		name = "resource-" + res.ID
	}
	name = sanitizeFilename(name)
	if !hasUsableFilenameExtension(name) {
		if ext := resourceFilenameExtension(res, contentType); ext != "" {
			name += ext
		}
	}
	return name
}

var usableFilenameExtensionPattern = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)

func hasUsableFilenameExtension(name string) bool {
	ext := strings.TrimSpace(strings.ToLower(filepath.Ext(name)))
	if ext == "" || ext == "." {
		return false
	}
	return usableFilenameExtensionPattern.MatchString(ext)
}

func resourceFilenameExtension(res moodle.Resource, contentType string) string {
	if ext := normalizeFilenameExtension(res.FileType); ext != "" {
		return ext
	}
	if ext := extensionFromContentType(contentType); ext != "" {
		return ext
	}
	if ext := extensionFromResourceURL(res.URL); ext != "" {
		return ext
	}
	return ""
}

func normalizeFilenameExtension(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, ".")
	if value == "" {
		return ""
	}
	return "." + value
}

func extensionFromResourceURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	ext := strings.TrimSpace(strings.ToLower(path.Ext(parsed.Path)))
	if ext == "" || ext == "." {
		return ""
	}
	switch ext {
	case ".php", ".asp", ".aspx", ".jsp", ".cgi":
		return ""
	}
	return ext
}

func extensionFromContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return ""
	}
	switch strings.ToLower(mediaType) {
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "application/json":
		return ".json"
	}
	exts, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(exts) == 0 {
		return ""
	}
	return exts[0]
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func sanitizeFilename(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return replacer.Replace(value)
}
