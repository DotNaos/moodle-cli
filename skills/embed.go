package skills

import "embed"

// RootDir is the path of the bundled skill set.
const RootDir = "moodle-services"

//go:embed moodle-services/* moodle-services/references/*
var FS embed.FS
