package skills

import "embed"

// RootDir is the path of the bundled skill set.
const RootDir = "moodle-cli"

//go:embed moodle-cli/* moodle-cli/references/*
var FS embed.FS
