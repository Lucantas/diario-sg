import { ActHit, EntityKind, OrganCount } from "./api";

export function entityPath(kind: EntityKind, slug: string) {
  return `/${kind}/${slug}`;
}

export function parseEntityPath(pathname: string): { kind: EntityKind; slug: string } | null {
  const m = pathname.match(/^\/(processo|contrato)\/([^/]+)$/);
  return m ? { kind: m[1] as EntityKind, slug: decodeURIComponent(m[2]) } : null;
}

export function groupByOrgan(acts: ActHit[], organs: OrganCount[], selected: string) {
  const byDate = [...acts].sort((a, b) => a.published_at.localeCompare(b.published_at) || a.id.localeCompare(b.id));
  return organs
    .filter((o) => selected === "" || o.organ === selected)
    .map((o) => ({ organ: o.organ, acts: byDate.filter((a) => (a.organ || "") === o.organ) }))
    .filter((g) => g.acts.length > 0);
}
