package domain

import "time"

type dispensaLimit struct {
	from  time.Time
	cents int64
}

type DispensaCategory string

const (
	DispensaGoods DispensaCategory = "compras"
	DispensaWorks DispensaCategory = "obras"
)

var (
	lei8666DispensaLimits = map[DispensaCategory][]dispensaLimit{
		DispensaGoods: {{civilDate(2018, 7, 19), 1760000}, {time.Time{}, 800000}},
		DispensaWorks: {{civilDate(2018, 7, 19), 3300000}, {time.Time{}, 1500000}},
	}
	lei14133DispensaLimits = map[DispensaCategory][]dispensaLimit{
		DispensaGoods: {
			{civilDate(2026, 1, 1), 6549211},
			{civilDate(2025, 1, 1), 6272559},
			{civilDate(2024, 1, 1), 5990602},
			{civilDate(2023, 1, 1), 5720833},
			{time.Time{}, 5000000},
		},
		DispensaWorks: {
			{civilDate(2026, 1, 1), 13098420},
			{civilDate(2025, 1, 1), 12545115},
			{civilDate(2024, 1, 1), 11981202},
			{civilDate(2023, 1, 1), 11441665},
			{time.Time{}, 10000000},
		},
	}
	lei14133Start = civilDate(2021, 4, 1)
	lei8666End    = civilDate(2023, 12, 30)
)

func DispensaLimitCents(published time.Time, citesLei14133 bool, category DispensaCategory) int64 {
	table := lei8666DispensaLimits[category]
	if !published.Before(lei8666End) || (citesLei14133 && !published.Before(lei14133Start)) {
		table = lei14133DispensaLimits[category]
	}
	for _, l := range table {
		if !published.Before(l.from) {
			return l.cents
		}
	}
	return 0
}

func civilDate(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
