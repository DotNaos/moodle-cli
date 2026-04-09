package moodle

import "regexp"

type CategoryFilterMode string

const (
	CategoryWhitelist CategoryFilterMode = "whitelist"
	CategoryBlacklist CategoryFilterMode = "blacklist"
)

type CategoryFilter struct {
	Mode     CategoryFilterMode
	Patterns []*regexp.Regexp
}

type SchoolSelectors struct {
	Username string
	Password string
	Submit   string
}

type SchoolConfig struct {
	ID                 string
	Name               string
	Active             bool
	MoodleURL          string
	LoginURL           string
	Selectors          SchoolSelectors
	CourseNamePatterns []*regexp.Regexp
	CategoryFilter     *CategoryFilter
}

const ActiveSchoolID = "fhgr"

// Multi-school support is not active yet.
// Keep additional school entries disabled so they cannot be used by accident.
var Schools = []SchoolConfig{
	{
		ID:        "fhgr",
		Name:      "FHGR",
		Active:    true,
		MoodleURL: "https://moodle.fhgr.ch",
		LoginURL:  "https://moodle.fhgr.ch/login/index.php",
		Selectors: SchoolSelectors{
			Username: `input[name="username"], input[name="login"], input[type="email"], input#username, input#login`,
			Password: `input[name="password"], input[type="password"], input#password`,
			Submit:   `button[type="submit"], input[type="submit"], button#login, button.btn-primary`,
		},
		CourseNamePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^\d{4}\s+(FS|HS)\s+FHGR\s+(\w+\s+)?`),
		},
		CategoryFilter: &CategoryFilter{
			Mode:     CategoryWhitelist,
			Patterns: []*regexp.Regexp{regexp.MustCompile(`^(FS|HS)\d{2}$`)},
		},
	},
	{
		ID:        "phgr",
		Name:      "PHGR",
		Active:    false,
		MoodleURL: "https://moodle.phgr.ch",
		LoginURL:  "https://moodle.phgr.ch/Shibboleth.sso/Login?providerId=https%3A%2F%2Feduid.ch%2Fidp%2Fshibboleth&target=https%3A%2F%2Fmoodle.phgr.ch%2Fauth%2Fshibboleth%2Findex.php",
		Selectors: SchoolSelectors{
			Username: `input[type="email"], input[name="email"], input#email, input[name="j_username"], input#username`,
			Password: `input[type="password"], input[name="password"], input#password, input[name="j_password"]`,
			Submit:   `button[type="submit"], input[type="submit"], button.btn-primary, button.btn`,
		},
		CourseNamePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^\d{4}\s+(FS|HS)\s+PHGR\s+(\w+\s+)?`),
		},
	},
}

func GetSchool(id string) *SchoolConfig {
	for i := range Schools {
		if Schools[i].ID == id {
			return &Schools[i]
		}
	}
	return nil
}

func ActiveSchools() []SchoolConfig {
	out := make([]SchoolConfig, 0, len(Schools))
	for _, school := range Schools {
		if school.Active {
			out = append(out, school)
		}
	}
	return out
}

func GetDefaultSchool() SchoolConfig {
	for _, school := range Schools {
		if school.ID == ActiveSchoolID && school.Active {
			return school
		}
	}
	panic("active default school is not configured")
}

func ShouldIncludeCategory(category string, school SchoolConfig) bool {
	if school.CategoryFilter == nil {
		return true
	}
	matches := false
	for _, pattern := range school.CategoryFilter.Patterns {
		if pattern.MatchString(category) {
			matches = true
			break
		}
	}
	if school.CategoryFilter.Mode == CategoryWhitelist {
		return matches
	}
	return !matches
}
