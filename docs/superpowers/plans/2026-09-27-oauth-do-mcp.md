# OAuth do servidor MCP — plano

**Spec:** `docs/superpowers/specs/2026-09-27-oauth-do-mcp-design.md`

1. Domínio `oauth.go`: `OAuthClient`, `OAuthCode`, validação de
   `redirect_uri` (https ou loopback, sem fragmento), casamento de URI
   (loopback com qualquer porta), verificação PKCE `S256`, geração do
   código e do `client_id`. Testes.
2. Caso de uso `OAuth`: `Register`, `CheckAuthorization`, `Authorize`
   (grava o código), `Exchange` (consome o código e emite a chave pelo
   `APIKeys`). Porta `OAuthRepository`, repositório Postgres e migration
   024. Testes com repositório falso e de integração.
3. HTTP: metadados em `/.well-known/…`, `/oauth/register`,
   `/oauth/authorize` (página HTML de consentimento, GET e POST) e
   `/oauth/token`; limite por IP no registro e no token. `WWW-Authenticate`
   do MCP com `resource_metadata`. Nginx e Vite passam `/.well-known/`.
4. Teste de integração do fluxo inteiro: descoberta pelo 401, registro,
   autorização, troca e chamada ao MCP com o token.
5. Página `/mcp` explica o conector do claude.ai e do ChatGPT; README,
   roadmap e "Depois da entrega".
