# OAuth do servidor MCP

## Problema

Os conectores do claude.ai e do ChatGPT só aceitam servidor MCP com login
OAuth: não há onde colar um cabeçalho `Authorization`. Hoje o servidor só
aceita a chave anônima gerada em `/mcp`, então funciona no Claude Code, no
Claude Desktop (via `mcp-remote`) e no Cursor, mas não nos dois clientes
com mais gente.

## Ideia

O site não tem conta de usuário, e não vai passar a ter: a chave é
anônima de propósito (só limita e conta o uso). O OAuth aqui é só o
caminho padrão para o cliente pegar uma chave sem a pessoa copiar e colar.
O token de acesso **é** uma chave MCP comum, emitida pelo mesmo
`APIKeys.Issue`, com o mesmo limite, a mesma contagem de uso e a mesma
revogação (`make revoke-key`).

Segue a especificação de autorização do MCP (2025-06-18 e 2025-11-25):
metadados do recurso protegido (RFC 9728), metadados do servidor de
autorização (RFC 8414), registro dinâmico de cliente (RFC 7591), código de
autorização com PKCE `S256` (OAuth 2.1) e o parâmetro `resource`
(RFC 8707).

## Endereços

Com `PUBLIC_WEB_URL` = `https://site`:

- recurso: `https://site/api/mcp`;
- issuer: `https://site` (raiz, que é onde os clientes procuram primeiro);
- `GET /.well-known/oauth-protected-resource` e
  `/.well-known/oauth-protected-resource/api/mcp`: `resource`,
  `authorization_servers: [issuer]`, `bearer_methods_supported: [header]`;
- `GET /.well-known/oauth-authorization-server`: issuer, endpoints,
  `response_types_supported: [code]`, `grant_types_supported:
  [authorization_code]`, `code_challenge_methods_supported: [S256]`,
  `token_endpoint_auth_methods_supported: [none]`;
- `POST /api/oauth/register`, `GET` e `POST /api/oauth/authorize`,
  `POST /api/oauth/token`.

O nginx do site e o proxy do Vite passam `/.well-known/` para a API. O 401
do `/mcp` traz `WWW-Authenticate: Bearer resource_metadata="…"`.

## Fluxo

1. **Registro.** JSON com `redirect_uris` (1 a 5) e `client_name`
   opcional (até 100 caracteres). Cada URI é `https` ou `http` num
   endereço de loopback (`localhost`, `127.0.0.1`, `[::1]`), sem
   fragmento. Só cliente público: `token_endpoint_auth_method` volta
   sempre `none`. Devolve `client_id` aleatório. Limite por IP: 20 por
   hora.
2. **Autorização (GET).** Confere `client_id` e `redirect_uri`
   registrados (loopback casa com qualquer porta, RFC 8252). Se um dos
   dois falha, mostra o erro na página e não redireciona. Os demais erros
   (`response_type` diferente de `code`, sem `code_challenge`, método
   diferente de `S256`, `resource` de outro servidor) voltam ao
   `redirect_uri` com `error`. Tudo certo: página de consentimento em
   HTML, servida pela API, com o nome do cliente, o domínio para onde a
   pessoa volta e o que a chave dá (leitura do Diário, 60 chamadas por
   minuto, revogável). A página não pode ser posta em moldura
   (`frame-ancestors 'none'`).
3. **Autorização (POST).** O botão manda os mesmos parâmetros; a API
   confere tudo de novo, grava o código (hash SHA-256, cliente,
   `redirect_uri`, desafio PKCE, `resource`, validade de 10 minutos) e
   redireciona com `code`, `state` e `iss`. "Cancelar" volta com
   `error=access_denied`.
4. **Token.** `grant_type=authorization_code`, `code`, `redirect_uri`,
   `client_id`, `code_verifier` e `resource` opcional. O código é apagado
   ao ser lido (uso único) e tem de casar com cliente, URI, PKCE e
   validade. Emite uma chave e devolve `{"access_token", "token_type":
   "Bearer"}`, sem `expires_in` nem refresh token: a chave não expira,
   como a de `/mcp`. Erros no formato do RFC 6749 (`invalid_grant`,
   `invalid_request`, `unsupported_grant_type`, `invalid_target`).

## Por que o consentimento sem login é seguro o bastante

A chave não dá acesso a nada que não seja público e anônimo: qualquer um
gera uma em `/mcp`. Quem enganar alguém para autorizar ganha só uma chave
que conseguiria sozinho. O que importa é não vazar o código para outro
destino (URI exata, PKCE obrigatório, uso único, 10 minutos) e não deixar
o registro virar spam (limite por IP).

## Dados

Migration 024: `oauth_clients` (id, nome, `redirect_uris text[]`,
`created_at`) e `oauth_codes` (hash do código, cliente, URI, desafio,
recurso, `expires_at`). Códigos vencidos são apagados a cada código novo.

## Fora

- Client ID Metadata Documents (URL como `client_id`): exige a API buscar
  uma URL qualquer; claude.ai e ChatGPT fazem registro dinâmico.
- Escopos, refresh token e token que expira: a chave é uma só, de leitura.
- Conta de usuário.
