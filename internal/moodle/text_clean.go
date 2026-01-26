package moodle

import (
  "regexp"
  "strings"
)

var (
  reSoftHyphenBreak = regexp.MustCompile(`\x{00ad}\s*\n\s*`)
  reHyphenLineBreak = regexp.MustCompile(`-\s*\n\s*`)
  reMultiSpace      = regexp.MustCompile(`[ \t]{2,}`)
  reMultiBlank      = regexp.MustCompile(`\n{3,}`)
  reLeadingSpace    = regexp.MustCompile(`^\s+`)
  reTrailingSpace   = regexp.MustCompile(`\s+$`)
  rePageNumber      = regexp.MustCompile(`^\d+\s*/\s*\d+$`)
)

func CleanExtractedText(input string) string {
  s := strings.ReplaceAll(input, "\r\n", "\n")
  s = strings.ReplaceAll(s, "\r", "\n")
  s = reSoftHyphenBreak.ReplaceAllString(s, "")
  s = reHyphenLineBreak.ReplaceAllString(s, "")
  s = reMultiSpace.ReplaceAllString(s, " ")

  lines := strings.Split(s, "\n")
  out := make([]string, 0, len(lines))

  i := 0
  for i < len(lines) {
    line := reTrailingSpace.ReplaceAllString(reLeadingSpace.ReplaceAllString(lines[i], ""), "")
    if line == "" {
      out = append(out, "")
      i++
      continue
    }
    if rePageNumber.MatchString(line) {
      i++
      continue
    }

    // merge with next lines based on heuristics
    for i+1 < len(lines) {
      next := reTrailingSpace.ReplaceAllString(reLeadingSpace.ReplaceAllString(lines[i+1], ""), "")
      if next == "" {
        break
      }

      // drop standalone page numbers like "2 / 2"
      if rePageNumber.MatchString(next) {
        i++
        continue
      }

      last := line[len(line)-1]
      nextFirst := next[0]

      isLowerNext := nextFirst >= 'a' && nextFirst <= 'z'
      isUpperNext := nextFirst >= 'A' && nextFirst <= 'Z'
      isSingleChar := len(line) == 1
      isShort := len(line) <= 2
      isAllCaps := len(line) <= 6 && line == strings.ToUpper(line)

      if isSingleChar {
        line = line + next
        i++
        continue
      }

      if isAllCaps && isUpperNext {
        line = line + " " + next
        i++
        continue
      }

      if isShort && isLowerNext {
        line = line + next
        i++
        continue
      }

      if isLowerNext && !isSentenceEnd(last) {
        line = line + " " + next
        i++
        continue
      }

      break
    }

    out = append(out, line)
    i++
  }

  s = strings.Join(out, "\n")
  s = reMultiBlank.ReplaceAllString(s, "\n\n")
  s = strings.TrimSpace(s)
  return s
}

func isSentenceEnd(ch byte) bool {
  switch ch {
  case '.', '!', '?', ':', ';':
    return true
  default:
    return false
  }
}
