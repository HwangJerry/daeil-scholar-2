// migrate_dump_parser — parses a WEO_BOARDBBS SQL dump to extract LEGACY NOTICE posts
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Column indices in WEO_BOARDBBS rows (0-based, matching CREATE TABLE declaration order).
const (
	colSEQ           = 0
	colGATE          = 1
	colSUBJECT       = 14
	colCONTENTS      = 15
	colCONTENT_FORMAT = 17
	colOPEN_YN       = 33
	colCount         = 38
)

// parseDump reads the SQL dump at path and returns LEGACY NOTICE posts that are active.
func parseDump(path string) ([]legacyPost, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dump: %w", err)
	}
	defer f.Close()

	const maxBuf = 32 * 1024 * 1024 // 32 MB — covers large single-line INSERT statements
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxBuf), maxBuf)

	for scanner.Scan() {
		line := scanner.Text()
		const marker = " VALUES "
		idx := strings.Index(line, marker)
		if idx < 0 || !strings.HasPrefix(line, "INSERT INTO") {
			continue
		}

		valuesStr := strings.TrimSuffix(strings.TrimSpace(line[idx+len(marker):]), ";")
		rowStrs := splitRowTuples([]byte(valuesStr))

		var posts []legacyPost
		for _, rowStr := range rowStrs {
			cols := parseRowValues([]byte(rowStr))
			if len(cols) < colCount {
				continue
			}
			if cols[colGATE] != "NOTICE" ||
				cols[colCONTENT_FORMAT] != "LEGACY" ||
				cols[colOPEN_YN] != "Y" ||
				cols[colCONTENTS] == "" ||
				strings.ToUpper(cols[colCONTENTS]) == "NULL" {
				continue
			}
			seq, err := strconv.Atoi(cols[colSEQ])
			if err != nil {
				continue
			}
			posts = append(posts, legacyPost{
				SEQ:      seq,
				Subject:  cols[colSUBJECT],
				Contents: cols[colCONTENTS],
			})
		}
		return posts, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dump: %w", err)
	}
	return nil, fmt.Errorf("INSERT statement not found in dump file")
}

// splitRowTuples splits the VALUES clause into individual row content strings
// (without outer parentheses), respecting single-quoted strings and escaping.
func splitRowTuples(data []byte) []string {
	var rows []string
	inStr := false
	depth := 0
	start := -1

	for i := 0; i < len(data); i++ {
		b := data[i]
		if inStr {
			if b == '\\' && i+1 < len(data) {
				i++ // skip escaped char
			} else if b == '\'' {
				if i+1 < len(data) && data[i+1] == '\'' {
					i++ // doubled-quote escape — stay in string
				} else {
					inStr = false
				}
			}
		} else {
			switch b {
			case '\'':
				inStr = true
			case '(':
				depth++
				if depth == 1 {
					start = i + 1
				}
			case ')':
				depth--
				if depth == 0 && start >= 0 {
					rows = append(rows, string(data[start:i]))
					start = -1
				}
			}
		}
	}
	return rows
}

// parseRowValues splits a single row's comma-separated SQL values into strings.
// Quoted strings are unescaped; NULL becomes the empty string.
func parseRowValues(data []byte) []string {
	var values []string
	var buf []byte
	inStr := false

	flush := func() {
		if buf != nil {
			val := strings.TrimSpace(string(buf))
			if strings.ToUpper(val) == "NULL" {
				val = ""
			}
			values = append(values, val)
			buf = nil
		}
	}

	for i := 0; i < len(data); i++ {
		b := data[i]
		if inStr {
			switch {
			case b == '\\' && i+1 < len(data):
				i++
				next := data[i]
				switch next {
				case '\'':
					buf = append(buf, '\'')
				case '\\':
					buf = append(buf, '\\')
				case 'n':
					buf = append(buf, '\n')
				case 'r':
					buf = append(buf, '\r')
				case 't':
					buf = append(buf, '\t')
				default:
					buf = append(buf, '\\', next)
				}
			case b == '\'' && i+1 < len(data) && data[i+1] == '\'':
				buf = append(buf, '\'')
				i++ // doubled-quote escape
			case b == '\'':
				inStr = false
				values = append(values, string(buf))
				buf = nil
			default:
				buf = append(buf, b)
			}
		} else {
			switch b {
			case '\'':
				inStr = true
				buf = nil // start fresh for quoted string
			case ',':
				flush()
			case ' ', '\t', '\r', '\n':
				// skip whitespace outside strings (unquoted values have no internal spaces)
			default:
				buf = append(buf, b)
			}
		}
	}
	flush() // handle last unquoted value (e.g., NULL at end of row)
	return values
}
