package domain

import "time"

type GroupBy string

const (
	GroupByCNPJ     GroupBy = "cnpj"
	GroupByProcesso GroupBy = "processo"
	GroupByOrgan    GroupBy = "orgao"
	GroupByType     GroupBy = "tipo"

	DefaultGroupLimit = 20
	MaxGroupLimit     = 50
	GroupExamples     = 3
)

func (g GroupBy) Valid() bool {
	switch g {
	case GroupByCNPJ, GroupByProcesso, GroupByOrgan, GroupByType:
		return true
	}
	return false
}

func (g GroupBy) EntityKind() (EntityKind, bool) {
	switch g {
	case GroupByCNPJ:
		return EntityCNPJ, true
	case GroupByProcesso:
		return EntityProcesso, true
	}
	return "", false
}

type GroupQuery struct {
	Filter        ActFilter
	By            GroupBy
	IncludePublic bool
	Limit         int
}

func (q *GroupQuery) Normalize() error {
	if !q.By.Valid() {
		return ErrInvalidFilter
	}
	if err := q.Filter.Normalize(); err != nil {
		return err
	}
	if q.Limit <= 0 {
		q.Limit = DefaultGroupLimit
	}
	q.Limit = min(q.Limit, MaxGroupLimit)
	return nil
}

func (q GroupQuery) ExcludedKeys() []string {
	if q.By != GroupByCNPJ || q.IncludePublic {
		return []string{}
	}
	return append(PublicBodyCNPJs(), PublicBodyRoots()...)
}

type ActRef struct {
	GazetteID   string
	Position    int
	Title       string
	Source      string
	PublishedAt time.Time
}

type ActGroup struct {
	Key           string
	Acts          int
	First         time.Time
	Last          time.Time
	MaxValueCents int64
	Examples      []ActRef
}

type ActGroups struct {
	MatchedActs int
	Groups      []ActGroup
}
