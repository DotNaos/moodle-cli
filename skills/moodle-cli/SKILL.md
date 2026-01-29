---
name: moodle-cli
description: Use when handling Moodle CLI tasks: login, list courses/files, lecture timetable lookups, printing file contents, and course download/export operations.
---

# Study Moodle

## Overview

Use the local Moodle CLI to login, list courses, files, and export/download course materials. Read the CLI repo docs before running commands.

## Quick Start

1. Read `/Users/oli/projects/active/moodle-cli/README.md` for current capabilities and status.
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
    - `moodle courses --json`

### List files for a course

- Use when asked about course materials, handouts, slides, or file lists.
- Command:
    - `moodle files <course-id> --json`

### Print file contents

- Use when asked to extract text from a specific file (PDFs supported).
- Command:
    - `moodle print <file-id>`

### Timetable (lectures)

- Use when asked about lecture times or next week’s schedule (this does NOT show exam deadlines).
- Command:
    - `moodle timetable --json`
- Flags: `--days <n>`, `--next-week`, `--unique`

### Download or export course

- Use when asked to download all files or export a full course.
- Commands:
    - `moodle download course <id> --zip|--files`
    - `moodle export course <id> --format=folder|zip`

## Resources

### references/

- `moodle-cli.md`: Quick command and data-location reference for the CLI.
- `timetable.md`: Timetable command reference.
