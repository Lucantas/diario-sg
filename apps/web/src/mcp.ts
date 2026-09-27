const SERVER_NAME = "diario-sg";

export interface McpSnippets {
  claudeCode: string;
  claudeDesktop: string;
  cursor: string;
}

function config(server: object) {
  return JSON.stringify({ mcpServers: { [SERVER_NAME]: server } }, null, 2);
}

export function mcpServerUrl(origin: string): string {
  return `${origin.replace(/\/+$/, "")}/api/mcp`;
}

export function oauthSnippet(url: string): string {
  return `claude mcp add --transport http ${SERVER_NAME} ${url}`;
}

export function mcpSnippets(url: string, key: string): McpSnippets {
  const header = `Authorization: Bearer ${key}`;
  return {
    claudeCode: `claude mcp add --transport http ${SERVER_NAME} ${url} --header "${header}"`,
    claudeDesktop: config({ command: "npx", args: ["mcp-remote", url, "--header", header] }),
    cursor: config({ url, headers: { Authorization: `Bearer ${key}` } }),
  };
}
