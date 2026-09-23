package domain

import "strings"

type Organ struct {
	Acronym string
	Name    string
}

type OrganCount struct {
	Organ
	Acts int
}

func IsKnownOrgan(acronym string) bool {
	_, ok := organNames[acronym]
	return ok
}

func OrganName(acronym string) string { return organNames[acronym] }

func NormalizeOrgan(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return "", true
	}
	if !IsKnownOrgan(s) {
		return "", false
	}
	return s, true
}

var organNames = map[string]string{
	"CAISAN":       "",
	"CEC":          "",
	"CGGMSG":       "",
	"CGMSG":        "",
	"CMAS":         "",
	"CMC":          "",
	"CMDCA":        "",
	"CMDDM":        "",
	"CMDDMSG":      "",
	"CMDM":         "",
	"CME":          "",
	"CMEL":         "",
	"CMPA":         "",
	"CMS":          "",
	"CODEPE":       "",
	"COMAD":        "",
	"COMDAR":       "",
	"COMDEC":       "",
	"COMDEPISG":    "",
	"COMDESG":      "",
	"COMENQ":       "",
	"COMINSG":      "",
	"COMIRSG":      "",
	"COMMADS":      "",
	"COMPAD":       "",
	"COMSEA":       "",
	"COMSEP":       "",
	"CONCIDADES":   "",
	"CONDEC":       "",
	"CONGES":       "",
	"CONSAD":       "",
	"COPEDE":       "",
	"CORIM":        "",
	"CPAC":         "",
	"CPADAG":       "",
	"CPAPPF":       "",
	"CPIAD":        "",
	"CPPF":         "",
	"CPPFGMF":      "",
	"CPPFS":        "",
	"EDURSAN":      "",
	"FAELSG":       "",
	"FAESG":        "",
	"FAESLG":       "",
	"FASG":         "",
	"FIASG":        "",
	"FMAS":         "",
	"FMS":          "",
	"FMSSG":        "",
	"FUMIA":        "",
	"FUMPARJ":      "",
	"FUNASG":       "",
	"FUNCULTURA":   "",
	"FUNDESG":      "",
	"FUNPARJ":      "",
	"GM":           "",
	"GMSG":         "",
	"IPASG":        "",
	"IPIIBA":       "",
	"PGE":          "",
	"PGM":          "",
	"PROMEA":       "",
	"SAMSADC":      "",
	"SECGOV":       "",
	"SECHABITA":    "",
	"SEMA":         "",
	"SEMAD":        "",
	"SEMAIMPD":     "",
	"SEMANT":       "",
	"SEMAP":        "",
	"SEMAPPAP":     "",
	"SEMAS":        "",
	"SEMCI":        "",
	"SEMCOM":       "",
	"SEMCOMP":      "",
	"SEMCON":       "",
	"SEMCS":        "",
	"SEMDE":        "",
	"SEMDUR":       "",
	"SEMED":        "",
	"SEMEL":        "",
	"SEMFA":        "",
	"SEMGIPE":      "",
	"SEMGOV":       "",
	"SEMGOVCOM":    "",
	"SEMGOVCOMS":   "",
	"SEMGOVCON":    "",
	"SEMHAB":       "",
	"SEMHABITA":    "",
	"SEMIDC":       "",
	"SEMIMD":       "",
	"SEMIND":       "",
	"SEMINDECON":   "",
	"SEMIURB":      "",
	"SEMIURBCPARJ": "",
	"SEMIURME":     "",
	"SEMMA":        "",
	"SEMMADU":      "",
	"SEMMATRAN":    "",
	"SEMOP":        "",
	"SEMPAD":       "",
	"SEMPESCA":     "",
	"SEMPLAN":      "",
	"SEMPPE":       "",
	"SEMSA":        "",
	"SEMSAD":       "",
	"SEMSADC":      "",
	"SEMSEP":       "",
	"SEMTCUL":      "",
	"SEMTRAB":      "",
	"SEMTRAN":      "",
	"SEOP":         "",
	"SETURCUL":     "",
	"SMAP":         "",
	"SMAS":         "",
	"SMC":          "",
	"SMDS":         "",
	"SMDSHABIA":    "",
	"SMDSIA":       "",
	"SMPPE":        "",
	"SMSP":         "",
	"SMSS":         "",
	"SMTC":         "",
	"SSM":          "",
	"SUBCOMP":      "",
}
