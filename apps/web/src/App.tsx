import { useState } from "react";
import { confirmSubscription, unsubscribe } from "./api";
import { CompanyPage } from "./CompanyPage";
import { DataPage } from "./DataPage";
import { EntityPage } from "./EntityPage";
import { AgentsPage } from "./AgentsPage";
import { BillPage } from "./BillPage";
import { SiteFooter, SiteHeader } from "./Brand";
import { parseBillPath } from "./bills";
import { BillsPage } from "./BillsPage";
import { FederalPage } from "./FederalPage";
import { parseEntityPath } from "./entity";
import { McpPage } from "./McpPage";
import { PanelsPage } from "./PanelsPage";
import { PatternsPage } from "./PatternsPage";
import { SearchPage } from "./SearchPage";
import { StaffPage } from "./StaffPage";
import { TCEPage } from "./TCEPage";

export function App() {
  const page = routePage(window.location.pathname);
  return (
    <>
      {page.type !== SearchPage && <SiteHeader />}
      {page}
      <SiteFooter />
    </>
  );
}

function routePage(path: string) {
  const token = new URLSearchParams(window.location.search).get("token") ?? "";
  if (path === "/confirmar") return <TokenPage kind="confirm" token={token} />;
  if (path === "/cancelar") return <TokenPage kind="cancel" token={token} />;
  if (path === "/dados") return <DataPage />;
  if (path === "/mcp") return <McpPage />;
  if (path === "/padroes") return <PatternsPage />;
  if (path === "/paineis") return <PanelsPage />;
  if (path === "/pessoal") return <StaffPage />;
  if (path === "/tce") return <TCEPage />;
  if (path === "/federal") return <FederalPage />;
  if (path === "/agentes") return <AgentsPage />;
  if (path === "/proposicoes") return <BillsPage />;
  const bill = parseBillPath(path);
  if (bill) return <BillPage process={bill} />;
  const entity = parseEntityPath(path);
  if (entity) return <EntityPage kind={entity.kind} slug={entity.slug} />;
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
        setMessage(`Alerta confirmado. Você será avisado quando houver ato novo sobre ${s.subject}.`);
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
