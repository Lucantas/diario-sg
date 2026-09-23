package domain

import "sort"

var organVariantOf = map[string]string{
	"CODEPE":     "COPEDE",
	"COMINSG":    "COMIRSG",
	"CONDEC":     "COMDEC",
	"FAESLG":     "FAELSG",
	"FUNPARJ":    "FUMPARJ",
	"SAMSADC":    "SEMSADC",
	"SEMSAD":     "SEMSADC",
	"SEMIND":     "SEMIMD",
	"SMPPE":      "SEMPPE",
	"SEMGOVCON":  "SEMGOVCOM",
	"SEMGOVCOMS": "SEMGOVCOM",
	"SEMHABITA":  "SECHABITA",
	"SEMIDC":     "SEMINDECON",
	"CGGMSG":     "CGMSG",
	"FMSSG":      "FMS",
	"SMAS":       "SEMAS",
	"SMSP":       "SEMSEP",
	"SECGOV":     "SEMGOV",
}

func PrincipalOrgan(acronym string) string {
	if principal, ok := organVariantOf[acronym]; ok {
		return principal
	}
	return acronym
}

func OrganAcronyms(principal string) []string {
	out := []string{principal}
	for variant, p := range organVariantOf {
		if p == principal {
			out = append(out, variant)
		}
	}
	sort.Strings(out)
	return out
}
