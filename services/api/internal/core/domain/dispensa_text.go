package domain

import "regexp"

var (
	valueDispensaRe = regexp.MustCompile(`(?i)art(?:igo)?\.?\s*(?:24|75)\s*,?\s*(?:caput\s*,?\s*)?(?:inciso\s*|inc\.\s*)?II(?:[^IVX]|$)`)
	lei14133Re      = regexp.MustCompile(`14\.133`)
	emergencyRe     = regexp.MustCompile(`(?i)emerg[êe]nc|calamidade`)
	republicationRe = regexp.MustCompile(`(?i)republicad[oa]`)
)

func CitesValueDispensa(body string) bool { return valueDispensaRe.MatchString(body) }
func CitesLei14133(body string) bool      { return lei14133Re.MatchString(body) }
func CitesEmergency(body string) bool     { return emergencyRe.MatchString(body) }
func IsRepublication(body string) bool    { return republicationRe.MatchString(body) }
