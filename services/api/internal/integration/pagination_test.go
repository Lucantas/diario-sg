//go:build integration

package integration

import "testing"

func TestSearchPastTheLastPageKeepsTheTotal(t *testing.T) {
	srv := newInvestigatorServer(t)

	var res struct {
		Total int              `json:"total"`
		Items []map[string]any `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/acts?limit=2&offset=50", &res)

	if res.Total != 3 || len(res.Items) != 0 {
		t.Fatalf("página além do fim deveria vir vazia com o total real: %+v", res)
	}
}
