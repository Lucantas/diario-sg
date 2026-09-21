package pmsg

import (
	"net/url"
	"testing"
	"time"
)

func TestEditionURL(t *testing.T) {
	base, _ := url.Parse("https://do.pmsg.rj.gov.br/")
	got := EditionURL(base, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC))
	if want := "https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf"; got != want {
		t.Errorf("esperava %s, veio %s", want, got)
	}
}
