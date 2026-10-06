package http

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

var renderProxies = []netip.Prefix{
	netip.MustParsePrefix("172.64.0.0/13"),
	netip.MustParsePrefix("162.158.0.0/15"),
	netip.MustParsePrefix("104.16.0.0/13"),
	netip.MustParsePrefix("74.220.48.0/20"),
}

func TestClientKey(t *testing.T) {
	cases := []struct {
		name, xff string
		trusted   []netip.Prefix
		want      string
	}{
		{"sem cabeçalho usa a conexão", "", renderProxies, "10.0.0.1"},
		{"sem proxies confiáveis mantém o primeiro endereço", " 200.1.2.3 , 10.0.0.9", nil, "200.1.2.3"},
		{"direto na API ignora o endereço forjado",
			"11.11.11.11,187.16.87.174, 172.64.222.167, 10.25.18.179", renderProxies, "187.16.87.174"},
		{"pelo site pula o proxy do site estático",
			"187.16.87.174, 162.158.41.34, 162.158.41.34,74.220.48.216, 172.68.175.20, 10.26.201.98", renderProxies, "187.16.87.174"},
		{"pelo site com endereço forjado",
			"22.22.22.22,187.16.87.174, 104.22.160.42, 104.22.160.42,74.220.48.206, 162.159.114.62, 10.26.201.98",
			append(renderProxies, netip.MustParsePrefix("162.159.0.0/16")), "187.16.87.174"},
		{"todos confiáveis cai na conexão", "172.64.1.1, 10.0.0.9", renderProxies, "10.0.0.1"},
		{"entrada inválida à direita dos confiáveis vira a chave", "1.1.1.1, lixo, 10.0.0.9", renderProxies, "lixo"},
		{"ipv6 do cliente", "2804:14c::1, 172.64.0.9", renderProxies, "2804:14c::1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/reports", nil)
			r.RemoteAddr = "10.0.0.1:5555"
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := clientKey(r, c.trusted); got != c.want {
				t.Errorf("veio %q, queria %q", got, c.want)
			}
		})
	}
}
