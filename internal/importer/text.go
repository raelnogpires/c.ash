package importer

import (
	"bytes"
	"errors"
	"strings"
	"unicode/utf8"

	"c.ash/internal/domain"
	"golang.org/x/text/encoding/charmap"
)

func decodeStatementText(data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxStatementSize {
		return "", domain.ErrInvalidStatement
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if utf8.Valid(data) {
		return strings.TrimPrefix(string(data), "\ufeff"), nil
	}
	decoded, err := charmap.Windows1252.NewDecoder().Bytes(data)
	if err != nil || !utf8.Valid(decoded) {
		return "", errors.Join(domain.ErrInvalidStatement, err)
	}
	return string(decoded), nil
}
