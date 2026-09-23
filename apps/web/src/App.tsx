import { useState } from "react";
import { confirmSubscription, unsubscribe } from "./api";
import { CompanyPage } from "./CompanyPage";
import { DataPage } from "./DataPage";
import { McpPage } from "./McpPage";
import { SearchPage } from "./SearchPage";

export function App() {
  const path = window.location.pathname;
  const token = new URLSearchParams(window.location.search).get("token") ?? "";
  if (path === "/confirmar") return <TokenPage kind="confirm" token={token} />;
  if (path === "/cancelar") return <TokenPage kind="cancel" token={token} />;
  if (path === "/dados") return <DataPage />;
  if (path === "/mcp") return <McpPage />;
  const company = path.match(/^\/empresa\/([\d./-]+)$/);
  if (company) return <CompanyPage cnpj={decodeURIComponent(company[1])} />;
  return <SearchPage />;
}

function TokenPage({ kind, token }: { kind: "confirm" | "cancel"; token: string }) {
  const [state, setState] = useState<"idle" | "done" | "error">("idle");
  const [message, setMessage] = useState("");
  const confirm = kind === "confirm";

  async function onClick() {
    try {
      if (confirm) {
        const s = await confirmSubscription(token);
        setMessage(`Alerta confirmado. Você será avisado quando “${s.query}” aparecer.`);
      } else {
        await unsubscribe(token);
        setMessage("Alerta cancelado. Você não receberá mais e-mails sobre ele.");
      }
      setState("done");
    } catch (err) {
      setMessage((err as Error).message);
      setState("error");
    }
  }

  return (
    <main className="page narrow">
      <h1>{confirm ? "Confirmar alerta" : "Cancelar alerta"}</h1>
      {!token && <p className="notice notice-error">Link incompleto. Abra o link do e-mail novamente.</p>}
      {token && state === "idle" && (
        <button className="primary" onClick={onClick}>
          {confirm ? "Confirmar alerta" : "Cancelar alerta"}
        </button>
      )}
      {state !== "idle" && (
        <p className={`notice ${state === "error" ? "notice-error" : ""}`}>{message}</p>
      )}
      <p><a href="/">Voltar para a busca</a></p>
    </main>
  );
}
