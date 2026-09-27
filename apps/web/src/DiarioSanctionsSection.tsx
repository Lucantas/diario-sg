import { ActHit, DiarioSanctionKind } from "./api";
import { Result } from "./components";

const KIND_LABEL: Record<DiarioSanctionKind, string> = {
  advertencia: "Advertência",
  multa: "Multa",
  suspensao: "Suspensão",
  impedimento: "Impedimento de licitar e contratar",
  inidoneidade: "Declaração de inidoneidade",
};

export function DiarioSanctionsSection({ sanctions }: { sanctions: { kind: DiarioSanctionKind; act: ActHit }[] }) {
  if (sanctions.length === 0) return null;
  return (
    <section className="results" aria-labelledby="diario-sanctions-heading">
      <h2 id="diario-sanctions-heading" className="panel-heading">Punições publicadas no Diário</h2>
      <p className="count">
        {sanctions.map((s) => KIND_LABEL[s.kind]).join(" · ")}. Lidas do título e da decisão de cada ato, na
        ordem abaixo. Quando a punição saiu colada ao ato anterior na página, o título é o do outro ato e a decisão está no
        fim do texto; confira na edição original.
      </p>
      <ol className="timeline">
        {sanctions.map((s) => <Result key={s.act.id} hit={s.act} />)}
      </ol>
    </section>
  );
}
