package domain

import (
	"testing"
	"time"
)

func TestCitesValueDispensa(t *testing.T) {
	yes := []string{
		"baseada no art. 24, inciso II da Lei Federal nº. 8.666",
		"fundamento legal o art. 24, inciso II c/c 23, II da Lei nº 8.666/93",
		"conforme artigo 75, inciso II da Lei de Licitações n.º 14.133",
		"nos termos do art. 75, II, da Lei 14.133",
		"Art. 75, caput, inciso II",
	}
	no := []string{
		"art. 24, inciso XIII, da Lei Federal nº 8.666/93",
		"nos termos do artigo 24, inciso XXII, da Lei 8.666",
		"art. 75, inciso III",
		"art. 75, inciso VIII",
		"Dispensa de Licitação",
	}
	for _, s := range yes {
		if !CitesValueDispensa(s) {
			t.Errorf("deveria citar dispensa por valor: %q", s)
		}
	}
	for _, s := range no {
		if CitesValueDispensa(s) {
			t.Errorf("não deveria citar dispensa por valor: %q", s)
		}
	}
}

func TestDispensaTextFlags(t *testing.T) {
	if !CitesLei14133("Lei Federal nº 14.133/2021") || CitesLei14133("Lei 8.666") {
		t.Error("CitesLei14133")
	}
	if !CitesEmergency("contratação emergencial") || !CitesEmergency("situação de EMERGÊNCIA") || !CitesEmergency("calamidade pública") || CitesEmergency("aquisição de vidros") {
		t.Error("CitesEmergency")
	}
	if !IsRepublication("Republicado por incorreção da PMSG.") || IsRepublication("publicado em 20/05") {
		t.Error("IsRepublication")
	}
}

func TestDispensaLimitCents(t *testing.T) {
	cases := []struct {
		day   time.Time
		cites bool
		want  int64
	}{
		{civilDate(2016, 5, 1), false, 800000},
		{civilDate(2018, 7, 18), false, 800000},
		{civilDate(2018, 7, 19), false, 1760000},
		{civilDate(2021, 3, 31), true, 1760000},
		{civilDate(2021, 4, 1), true, 5000000},
		{civilDate(2022, 6, 1), false, 1760000},
		{civilDate(2023, 6, 1), true, 5720833},
		{civilDate(2023, 12, 29), false, 1760000},
		{civilDate(2023, 12, 30), false, 5720833},
		{civilDate(2024, 3, 1), false, 5990602},
		{civilDate(2025, 3, 1), false, 6272559},
		{civilDate(2026, 3, 1), false, 6549211},
	}
	for _, c := range cases {
		if got := DispensaLimitCents(c.day, c.cites); got != c.want {
			t.Errorf("%s (14.133=%v): %d, esperava %d", c.day.Format("2006-01-02"), c.cites, got, c.want)
		}
	}
}

func TestFormatBRL(t *testing.T) {
	for cents, want := range map[int64]string{0: "R$ 0,00", 5: "R$ 0,05", 1186200: "R$ 11.862,00", 2891160: "R$ 28.911,60", 654921100: "R$ 6.549.211,00"} {
		if got := FormatBRL(cents); got != want {
			t.Errorf("FormatBRL(%d) = %q, esperava %q", cents, got, want)
		}
	}
}
