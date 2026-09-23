import { FormEvent, useState } from "react";
import { ActHit, ReportKind, reportError } from "./api";

const KINDS: [ReportKind, string][] = [
  ["texto_errado", "O texto está errado ou misturado com outro ato"],
  ["tipo_errado", "O tipo do ato está errado"],
  ["orgao_errado", "O órgão está errado"],
  ["pagina_errada", "A página indicada está errada"],
  ["outro", "Outro problema"],
];

export const REPORT_MESSAGE_MAX = 2000;

type Status = { state: "editing" | "sending" | "sent" } | { state: "failed"; error: string };

export function ReportForm({ hit }: { hit: ActHit }) {
  const [kind, setKind] = useState<ReportKind>("texto_errado");
  const [message, setMessage] = useState("");
  const [website, setWebsite] = useState("");
  const [status, setStatus] = useState<Status>({ state: "editing" });

  async function submit(e: FormEvent) {
    e.preventDefault();
    setStatus({ state: "sending" });
    try {
      await reportError({ gazette_id: hit.gazette_id, position: hit.position, act_title: hit.title, kind, message, website });
      setStatus({ state: "sent" });
    } catch (err) {
      setStatus({ state: "failed", error: err instanceof Error ? err.message : "Não foi possível enviar." });
    }
  }

  if (status.state === "sent") {
    return (
      <p className="report report-sent" role="status">
        Obrigado. Vamos conferir este ato no PDF original; quando a extração for corrigida, a busca passa a mostrar o
        texto certo.
      </p>
    );
  }

  const fieldId = `report-${hit.id}`;
  return (
    <form className="report" onSubmit={submit}>
      <label htmlFor={`${fieldId}-kind`}>O que está errado?</label>
      <select id={`${fieldId}-kind`} value={kind} onChange={(e) => setKind(e.target.value as ReportKind)}>
        {KINDS.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
      </select>
      <label htmlFor={`${fieldId}-message`}>{kind === "outro" ? "Descreva o problema" : "Detalhes (opcional)"}</label>
      <textarea
        id={`${fieldId}-message`}
        rows={3}
        maxLength={REPORT_MESSAGE_MAX}
        required={kind === "outro"}
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        placeholder="Por exemplo: o texto deste ato continua no ato seguinte."
      />
      <input
        className="trap"
        name="website"
        aria-hidden="true"
        tabIndex={-1}
        autoComplete="off"
        value={website}
        onChange={(e) => setWebsite(e.target.value)}
      />
      <p className="report-actions">
        <button type="submit" disabled={status.state === "sending"}>
          {status.state === "sending" ? "Enviando…" : "Enviar reporte"}
        </button>
        <span className="fineprint">Não pedimos e-mail nem guardamos quem enviou.</span>
      </p>
      {status.state === "failed" && <p className="notice notice-error" role="alert">{status.error}</p>}
    </form>
  );
}
