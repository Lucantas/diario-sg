import { FormEvent, useState } from "react";
import { IssuedKey, issueMcpKey, revokeMcpKey } from "./api";
import { mcpSnippets } from "./mcp";

const TOOLS: [string, string][] = [
  ["buscar_atos", "busca nos Diários da Prefeitura e da Câmara com os mesmos filtros do site"],
  ["ler_ato", "texto completo de um ato, com citação pronta"],
  ["entidade", "atos que citam um CNPJ, processo ou contrato, com contagem por tipo e soma dos valores citados"],
  ["agrupar", "conta os atos encontrados por CNPJ, processo, órgão ou tipo, sem os CNPJs de órgãos públicos"],
  ["fontes", "período coberto, última coleta e lacunas conhecidas"],
];

export function McpPage() {
  const [issued, setIssued] = useState<IssuedKey | null>(null);
  const [issueError, setIssueError] = useState("");
  const [website, setWebsite] = useState("");
  const [sending, setSending] = useState(false);

  async function onIssue() {
    setSending(true);
    setIssueError("");
    try {
      setIssued(await issueMcpKey(website));
    } catch (err) {
      setIssueError((err as Error).message);
    } finally {
      setSending(false);
    }
  }

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Para usar com IA</p>
        <h1>Pergunte ao Diário pela sua IA.</h1>
        <p className="lede">
          O Diário SG tem um servidor MCP: você pluga na IA que já usa (Claude, Cursor e outras compatíveis) e
          pergunta em linguagem natural. A IA busca e lê os atos por aqui e cita a edição, a página e a cópia
          arquivada. A IA é sua; nós só servimos os dados.
        </p>
      </header>

      <section className="howto" aria-label="Ferramentas">
        <h2>O que a IA consegue consultar</h2>
        <ul className="tools">
          {TOOLS.map(([name, text]) => (
            <li key={name}><code>{name}</code>: {text}</li>
          ))}
        </ul>
      </section>

      <section className="howto" aria-label="Gerar chave">
        <h2>1. Gere a sua chave</h2>
        <p>
          A chave não pede cadastro nem e-mail. Ela serve para contar o uso e aplicar o limite de 60 chamadas por
          minuto. Ela aparece uma única vez: guarde num lugar seguro.
        </p>
        <input className="trap" aria-hidden="true" tabIndex={-1} autoComplete="off" name="website"
          value={website} onChange={(e) => setWebsite(e.target.value)} />
        {!issued && (
          <button className="primary" onClick={onIssue} disabled={sending}>
            {sending ? "Gerando…" : "Gerar chave"}
          </button>
        )}
        {issueError && <p className="notice notice-error">{issueError}</p>}
        {issued && <IssuedKeyPanel issued={issued} />}
      </section>

      <RevokeForm />

      <p className="fineprint">
        Os atos são públicos, mas citam pessoas de verdade. A IA recebe as mesmas regras do site: cite a edição
        original, que é a que vale, e não monte perfis de pessoas físicas a partir dos atos.
      </p>
    </main>
  );
}

function IssuedKeyPanel({ issued }: { issued: IssuedKey }) {
  const snippets = mcpSnippets(issued.mcp_url, issued.key);
  const [copied, setCopied] = useState(false);

  async function copy() {
    await navigator.clipboard.writeText(issued.key);
    setCopied(true);
  }

  return (
    <>
      <div className="cite">
        <p>{issued.key}</p>
        <button onClick={copy}>{copied ? "Copiada" : "Copiar chave"}</button>
      </div>
      <h2>2. Configure a sua IA</h2>
      <p>Claude Code, no terminal:</p>
      <pre>{snippets.claudeCode}</pre>
      <p>Claude Desktop, em <code>claude_desktop_config.json</code> (precisa do Node instalado):</p>
      <pre>{snippets.claudeDesktop}</pre>
      <p>Cursor, em <code>.cursor/mcp.json</code>:</p>
      <pre>{snippets.cursor}</pre>
      <p className="hint">
        Outras IAs: servidor HTTP em <code>{issued.mcp_url}</code> com o cabeçalho{" "}
        <code>Authorization: Bearer</code> e a chave. Os conectores do claude.ai e do ChatGPT pedem login por
        OAuth, que ainda não temos.
      </p>
    </>
  );
}

function RevokeForm() {
  const [key, setKey] = useState("");
  const [state, setState] = useState<"idle" | "done" | "error">("idle");
  const [message, setMessage] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      await revokeMcpKey(key);
      setState("done");
      setMessage("Chave revogada. Quem tiver essa chave não consegue mais usar o servidor.");
      setKey("");
    } catch (err) {
      setState("error");
      setMessage((err as Error).message);
    }
  }

  return (
    <section className="howto" aria-label="Revogar chave">
      <h2>Vazou a chave?</h2>
      <form className="search" onSubmit={onSubmit}>
        <label className="visually-hidden" htmlFor="revoke-key">Chave a revogar</label>
        <input id="revoke-key" value={key} onChange={(e) => setKey(e.target.value)} placeholder="dsg_…"
          autoComplete="off" required />
        <button type="submit">Revogar</button>
      </form>
      {state !== "idle" && <p className={`notice ${state === "error" ? "notice-error" : ""}`}>{message}</p>}
    </section>
  );
}
