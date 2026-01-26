package moodle

import (
  "bytes"

  pdf "github.com/ledongthuc/pdf"
)

func ExtractPDFText(data []byte) (string, error) {
  reader := bytes.NewReader(data)
  r, err := pdf.NewReader(reader, int64(len(data)))
  if err != nil {
    return "", err
  }
  var buf bytes.Buffer
  b, err := r.GetPlainText()
  if err != nil {
    return "", err
  }
  if _, err := buf.ReadFrom(b); err != nil {
    return "", err
  }
  return buf.String(), nil
}
