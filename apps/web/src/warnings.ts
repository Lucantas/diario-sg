import { ActHit } from "./api";

type PageRange = Pick<ActHit, "page_start" | "page_end">;

const TEXTS = new Map<string, (hit: PageRange) => string>([
  ["sem_numero", () => "O título deste ato não tem número. A extração pode ter juntado ou cortado atos; confira no PDF."],
  ["so_titulo", () => "Só o título deste ato foi extraído. O texto dele está no PDF."],
  ["muitas_paginas", (hit) =>
    `Este ato ocupa ${(hit.page_end ?? 0) - (hit.page_start ?? 0) + 1} páginas. Ele pode ter engolido atos vizinhos; confira no PDF.`],
]);

export function warningText(code: string, hit: PageRange) {
  return TEXTS.get(code)?.(hit) ?? "";
}
