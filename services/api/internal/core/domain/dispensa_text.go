package domain

import "regexp"

var (
	lei14133Re      = regexp.MustCompile(`14\.133`)
	emergencyRe     = regexp.MustCompile(`(?i)emerg[êe]nc|calamidade`)
	republicationRe = regexp.MustCompile(`(?i)republicad[oa]`)
)

func CitesValueDispensa(body string) bool { return citesAny(body, Art24II, Art75II) }
func CitesWorksDispensa(body string) bool { return citesAny(body, Art24I, Art75I) }
func CitesLei14133(body string) bool      { return lei14133Re.MatchString(body) }
func CitesEmergency(body string) bool     { return emergencyRe.MatchString(body) }
func IsRepublication(body string) bool    { return republicationRe.MatchString(body) }
