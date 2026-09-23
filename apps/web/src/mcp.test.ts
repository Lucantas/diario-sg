import { describe, expect, it } from "vitest";
import { mcpSnippets } from "./mcp";

const url = "https://diario.exemplo/api/mcp";
const key = "dsg_abc123";

describe("mcpSnippets", () => {
  it("monta o comando do Claude Code com a URL e a chave", () => {
    expect(mcpSnippets(url, key).claudeCode).toBe(
      `claude mcp add --transport http diario-sg ${url} --header "Authorization: Bearer ${key}"`,
    );
  });

  it("usa o mcp-remote no Claude Desktop", () => {
    const config = JSON.parse(mcpSnippets(url, key).claudeDesktop);

    expect(config.mcpServers["diario-sg"]).toEqual({
      command: "npx",
      args: ["mcp-remote", url, "--header", `Authorization: Bearer ${key}`],
    });
  });

  it("passa URL e cabeçalho no Cursor", () => {
    const config = JSON.parse(mcpSnippets(url, key).cursor);

    expect(config.mcpServers["diario-sg"]).toEqual({ url, headers: { Authorization: `Bearer ${key}` } });
  });
});
