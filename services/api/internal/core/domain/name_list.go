package domain

import (
	"regexp"
	"strings"
)

const NameListMinLines = 50

var nameLine = regexp.MustCompile(`(?m)^\p{Lu}{2,}(?: \p{Lu}{2,}){1,6}[ \t\r]*$`)

func NameLines(body string) int {
	return len(nameLine.FindAllStringIndex(body, -1))
}

func (f ActFilter) TextQuery() string {
	if f.Name == "" {
		return f.Query
	}
	return strings.TrimSpace(f.Query + ` "` + f.Name + `"`)
}

func (f ActFilter) ExcludesNameLists() bool {
	return f.Name != "" && !f.IncludeLists
}

func cleanName(name string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(name, `"`, " ")), " ")
}
