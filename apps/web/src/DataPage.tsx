import { useEffect, useState } from "react";
import { formatBytes } from "./format";

interface DumpFile {
  name: string;
  rows: number;
  bytes: number;
  sha256: string;
}

interface DumpManifest {
  generated_at: string;
  files: DumpFile[];
}

const DESCRIPTIONS: Record<string, string> = {
  "gazettes.csv.gz": "Uma linha por edição: número, data, se é extra, link do PDF e SHA-256.",
  "acts.csv.gz": "Uma linha por ato: edição, posição, tipo, órgão, título, páginas e texto completo.",
  "act_entities.csv.gz": "CNPJs, valores (em centavos), contratos e processos extraídos de cada ato.",
};

export function DataPage() {
  const [manifest, setManifest] = useState<DumpManifest | null>(null);
  const [missing, setMissing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetch("/dados/manifest.json")
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error(String(res.status)))))
      .then((m: DumpManifest) => { if (!cancelled) setManifest(m); })
      .catch(() => { if (!cancelled) setMissing(true); });
    return () => { cancelled = true; };
  }, []);

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Dados abertos</p>
        <h1>A base inteira, para baixar.</h1>
        <p className="lede">
          Toda semana o Diário SG publica tudo o que extraiu do Diário Oficial: edições, atos com o texto
          completo e os CNPJs e valores encontrados. Em CSV compactado, para abrir em qualquer ferramenta.
        </p>
      </header>

      {missing && <p className="notice">O dump ainda não foi publicado. Ele é gerado uma vez por semana.</p>}
      {!missing && manifest === null && <p className="count">Carregando…</p>}

      {manifest && (
        <section aria-label="Arquivos">
          <p className="count">
            Gerado em {new Date(manifest.generated_at).toLocaleString("pt-BR", { dateStyle: "long", timeStyle: "short" })}.
          </p>
          <ul className="files">
            {manifest.files.map((f) => (
              <li key={f.name}>
                <p className="file-name">
                  <a href={`/dados/${f.name}`} download>{f.name}</a>
                  <span>{formatBytes(f.bytes)} · {f.rows.toLocaleString("pt-BR")} linhas</span>
                </p>
                {DESCRIPTIONS[f.name] && <p>{DESCRIPTIONS[f.name]}</p>}
                <p className="hash">SHA-256 {f.sha256}</p>
              </li>
            ))}
          </ul>
          <p>
            As colunas de cada arquivo estão descritas no <a href="/dados/LEIAME.txt">LEIAME.txt</a>.
            O <a href="/dados/manifest.json">manifest.json</a> traz os mesmos números para conferência automática.
          </p>
        </section>
      )}

      <section className="howto" aria-label="Como abrir">
        <h2>Como abrir</h2>
        <p>Em Python, com pandas:</p>
        <pre>{`import pandas as pd
atos = pd.read_csv("acts.csv.gz")`}</pre>
        <p>Num banco SQLite, para explorar com o Datasette:</p>
        <pre>{`gunzip -k acts.csv.gz gazettes.csv.gz act_entities.csv.gz
sqlite-utils insert diario.db acts acts.csv --csv
sqlite-utils insert diario.db gazettes gazettes.csv --csv
sqlite-utils insert diario.db act_entities act_entities.csv --csv
datasette diario.db`}</pre>
        <p>
          Para ligar um ato entre dumps diferentes, use a edição e a posição (<code>gazette_id</code> e{" "}
          <code>position</code>): o <code>id</code> do ato muda quando a edição é reprocessada.
        </p>
      </section>

      <p className="fineprint">
        Os atos são públicos, mas citam pessoas de verdade. A LGPD vale para quem reutiliza: não monte perfis de
        pessoas físicas a partir destes dados e cite sempre a edição original, que é a que vale.
      </p>
    </main>
  );
}
