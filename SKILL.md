---
name: moodle-cli
description: "Use when handling Moodle CLI tasks: login, list courses/files, lecture timetable lookups, printing file contents, and course download/export operations."
---

# Study Moodle

## Overview

Use the local Moodle CLI to login, list courses, files, and export/download course materials. Read the CLI repo docs before running commands.

## Quick Start

1. Read `README.md` for current capabilities and status.
2. Run the CLI as `moodle` (installed on PATH; use `source ~/.zshrc` first if needed).
3. Prefer JSON outputs (`--json`) when available and parse results for the user.

## Core Tasks

### Login

- Use when a request requires authenticated access or commands fail with session expired.
- Command:
    - `moodle login`

### List courses

- Use when asked about enrolled courses, course IDs, or to confirm a course exists.
- Command:
    - `moodle list courses --json`

### List files for a course

- Use when asked about course materials, handouts, slides, or file lists.
- Command:
    - `moodle list files <course-id|name> --json`

### Print file contents

- Use when asked to extract text from a specific file (PDFs supported).
- Command:
    - `moodle print course <course-id|name> <resource-id|name>`

### Timetable (lectures)

- Use when asked about lecture times or next week’s schedule (this does NOT show exam deadlines).
- Command:
    - `moodle list timetable --json`
- Flags: `--days <n>`, `--next-week`, `--unique`

### Download or export course

- Use when asked to download all files or export a full course.
- Commands:
    - `moodle download file <course-id|name> <resource-id|name> --output-dir <path>`
    - `moodle download file <course-id|name> --all --output-dir <path>`
    - `moodle export course <course-id|name> --output-dir <path>`

## Resources

### references/

- `skills/moodle-cli/references/moodle-cli.md`: Quick command and data-location reference for the CLI.
- `skills/moodle-cli/references/timetable.md`: Timetable command reference.
