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
		"Art. 75, caput, inciso II da Lei nº 14.133/2021",
		"nos termos do Inciso II do Artigo 75 da Lei Federal Nº14.133/2021",
		"nos termos do inciso II do artigo 24 da Lei Federal nº 8.666/1993",
	}
	no := []string{
		"art. 24, inciso XIII, da Lei Federal nº 8.666/93",
		"nos termos do artigo 24, inciso XXII, da Lei 8.666",
		"art. 75, inciso III",
		"art. 75, inciso VIII",
		"Dispensa de Licitação",
		"inciso III do artigo 75",
		"inciso II do artigo 76",
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
		day      time.Time
		cites    bool
		category DispensaCategory
		want     int64
	}{
		{civilDate(2016, 5, 1), false, DispensaGoods, 800000},
		{civilDate(2018, 7, 18), false, DispensaGoods, 800000},
		{civilDate(2018, 7, 19), false, DispensaGoods, 1760000},
		{civilDate(2021, 3, 31), true, DispensaGoods, 1760000},
		{civilDate(2021, 4, 1), true, DispensaGoods, 5000000},
		{civilDate(2022, 6, 1), false, DispensaGoods, 1760000},
		{civilDate(2023, 6, 1), true, DispensaGoods, 5720833},
		{civilDate(2023, 12, 29), false, DispensaGoods, 1760000},
		{civilDate(2023, 12, 30), false, DispensaGoods, 5720833},
		{civilDate(2024, 3, 1), false, DispensaGoods, 5990602},
		{civilDate(2025, 3, 1), false, DispensaGoods, 6272559},
		{civilDate(2026, 3, 1), false, DispensaGoods, 6549211},
		{civilDate(2018, 7, 18), false, DispensaWorks, 1500000},
		{civilDate(2018, 7, 19), false, DispensaWorks, 3300000},
		{civilDate(2021, 4, 1), true, DispensaWorks, 10000000},
		{civilDate(2022, 6, 1), false, DispensaWorks, 3300000},
		{civilDate(2023, 6, 1), true, DispensaWorks, 11441665},
		{civilDate(2024, 3, 1), false, DispensaWorks, 11981202},
		{civilDate(2025, 3, 1), false, DispensaWorks, 12545115},
		{civilDate(2026, 3, 1), false, DispensaWorks, 13098420},
	}
	for _, c := range cases {
		if got := DispensaLimitCents(c.day, c.cites, c.category); got != c.want {
			t.Errorf("%s (14.133=%v, %s): %d, esperava %d", c.day.Format("2006-01-02"), c.cites, c.category, got, c.want)
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
