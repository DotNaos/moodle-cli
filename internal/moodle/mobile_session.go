package moodle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MobileSession struct {
	SchoolID     string    `json:"schoolId,omitempty"`
	SiteURL      string    `json:"siteUrl"`
	UserID       int       `json:"userId"`
	Token        string    `json:"token"`
	PrivateToken string    `json:"privateToken,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MobileClient struct {
	Session MobileSession
	School  SchoolConfig
	http    *http.Client
}

func MobileSessionFromToken(token MobileToken) MobileSession {
	return MobileSession{
		SiteURL:      token.SiteURL,
		UserID:       token.UserID,
		Token:        token.Token,
		PrivateToken: token.PrivateToken,
		CreatedAt:    time.Now(),
	}
}

func (s MobileSession) ResolvedSchoolID() string {
	if s.SchoolID != "" {
		return s.SchoolID
	}
	return ActiveSchoolID
}

func LoadMobileSession(path string) (MobileSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MobileSession{}, err
	}
	var session MobileSession
	if err := json.Unmarshal(data, &session); err != nil {
		return MobileSession{}, err
	}
	if session.SiteURL == "" {
		return MobileSession{}, fmt.Errorf("mobile session missing siteUrl")
	}
	if session.UserID == 0 {
		return MobileSession{}, fmt.Errorf("mobile session missing userId")
	}
	if session.Token == "" {
		return MobileSession{}, fmt.Errorf("mobile session missing token")
	}
	return session, nil
}

func SaveMobileSession(path string, session MobileSession) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func NewMobileClient(session MobileSession, schoolID string) (*MobileClient, error) {
	school, err := resolveSchool(schoolID)
	if err != nil {
		return nil, err
	}
	if session.SiteURL == "" || session.Token == "" || session.UserID == 0 {
		return nil, fmt.Errorf("mobile session is incomplete")
	}
	return &MobileClient{
		Session: session,
		School:  school,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (c *MobileClient) ValidateSession() error {
	_, err := c.FetchMobileSiteInfo()
	return err
}

func (c *MobileClient) FetchCourses() ([]Course, error) {
	var courses []MobileCourse
	values := url.Values{}
	values.Set("userid", strconv.Itoa(c.Session.UserID))
	if err := c.callMobileAPI("core_enrol_get_users_courses", values, &courses); err != nil {
		return nil, err
	}

	result := make([]Course, 0, len(courses))
	for _, course := range courses {
		result = append(result, Course{
			ID:        course.ID,
			Fullname:  DisplayCourseName(course.FullName, c.School.CourseNamePatterns),
			Shortname: course.ShortName,
			ViewURL:   strings.TrimRight(c.Session.SiteURL, "/") + "/course/view.php?id=" + strconv.Itoa(course.ID),
		})
	}
	return result, nil
}

func (c *MobileClient) FetchCourseResources(courseID string) ([]Resource, string, error) {
	var sections []mobileCourseSection
	values := url.Values{}
	values.Set("courseid", courseID)
	if err := c.callMobileAPI("core_course_get_contents", values, &sections); err != nil {
		return nil, "", err
	}

	resources := make([]Resource, 0)
	for _, section := range sections {
		sectionID := strconv.Itoa(section.ID)
		for _, module := range section.Modules {
			resource, ok := mobileModuleToResource(c.Session.SiteURL, courseID, sectionID, section.Name, module)
			if ok {
				resources = append(resources, resource)
			}
		}
	}
	return resources, "", nil
}

func (c *MobileClient) FetchMobileSiteInfo() (MobileSiteInfo, error) {
	var info MobileSiteInfo
	if err := c.callMobileAPI("core_webservice_get_site_info", nil, &info); err != nil {
		return MobileSiteInfo{}, err
	}
	return info, nil
}

func (c *MobileClient) callMobileAPI(function string, values url.Values, target any) error {
	token := MobileToken{
		SiteURL:      c.Session.SiteURL,
		UserID:       c.Session.UserID,
		Token:        c.Session.Token,
		PrivateToken: c.Session.PrivateToken,
	}
	client := &Client{http: c.http}
	return client.CallMobileAPI(token, function, values, target)
}

type mobileCourseSection struct {
	ID      int            `json:"id"`
	Name    string         `json:"name"`
	Visible int            `json:"visible"`
	Modules []mobileModule `json:"modules"`
}

type mobileModule struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	ModName  string          `json:"modname"`
	URL      string          `json:"url"`
	Visible  int             `json:"visible"`
	Contents []mobileContent `json:"contents"`
}

type mobileContent struct {
	Type     string `json:"type"`
	FileName string `json:"filename"`
	FilePath string `json:"filepath"`
	FileSize int    `json:"filesize"`
	FileURL  string `json:"fileurl"`
}

func mobileModuleToResource(siteURL string, courseID string, sectionID string, sectionName string, module mobileModule) (Resource, bool) {
	if module.ModName == "label" || module.ID == 0 {
		return Resource{}, false
	}

	resourceType := "resource"
	id := strconv.Itoa(module.ID)
	if module.ModName == "folder" {
		resourceType = "folder"
		id = "folder-" + id
	}

	fileType := inferMobileFileType(module)
	return Resource{
		ID:          id,
		Name:        strings.TrimSpace(module.Name),
		URL:         firstNonEmpty(module.URL, strings.TrimRight(siteURL, "/")+"/mod/"+module.ModName+"/view.php?id="+strconv.Itoa(module.ID)),
		Type:        resourceType,
		CourseID:    courseID,
		SectionID:   sectionID,
		SectionName: sectionName,
		FileType:    fileType,
	}, true
}

func inferMobileFileType(module mobileModule) string {
	for _, content := range module.Contents {
		if content.FileName == "" {
			continue
		}
		name := strings.ToLower(content.FileName)
		switch {
		case strings.HasSuffix(name, ".pdf"):
			return "pdf"
		case strings.HasSuffix(name, ".doc") || strings.HasSuffix(name, ".docx"):
			return "docx"
		case strings.HasSuffix(name, ".xls") || strings.HasSuffix(name, ".xlsx"):
			return "xlsx"
		case strings.HasSuffix(name, ".ppt") || strings.HasSuffix(name, ".pptx"):
			return "pptx"
		case strings.HasSuffix(name, ".zip"):
			return "zip"
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
