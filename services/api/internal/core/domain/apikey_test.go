package domain

import (
	"strings"
	"testing"
)

func TestNewAPIKeySecretIsPrefixedAndUnique(t *testing.T) {
	a, err := NewAPIKeySecret()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewAPIKeySecret()
	if !strings.HasPrefix(a, "dsg_") || len(a) != len("dsg_")+43 {
		t.Fatalf("segredo fora do formato: %q", a)
	}
	if a == b {
		t.Fatal("dois segredos iguais")
	}
}

func TestHashAPIKeyIsStableHex(t *testing.T) {
	if HashAPIKey("dsg_abc") != HashAPIKey("dsg_abc") || len(HashAPIKey("dsg_abc")) != 64 {
		t.Fatal("hash deveria ser SHA-256 em hex e estável")
	}
	if HashAPIKey("dsg_abc") == HashAPIKey("dsg_abd") {
		t.Fatal("segredos diferentes, hashes diferentes")
	}
}

func TestAPIKeyPrefixTakesEightCharactersAfterTheMarker(t *testing.T) {
	if got := APIKeyPrefix("dsg_ABCDEFGHijkl"); got != "ABCDEFGH" {
		t.Fatalf("veio %q", got)
	}
	if got := APIKeyPrefix("curta"); got != "" {
		t.Fatalf("segredo fora do formato não tem prefixo, veio %q", got)
	}
}
