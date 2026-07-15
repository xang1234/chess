package domain

import (
	"sort"
	"strings"
)

func NormalizeThemes(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized != "" {
			unique[normalized] = struct{}{}
		}
	}

	themes := make([]string, 0, len(unique))
	for theme := range unique {
		themes = append(themes, theme)
	}
	sort.Strings(themes)
	return themes
}
