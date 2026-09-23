
export type ActType =
  | "nomeacao" | "exoneracao" | "contrato" | "aditivo" | "licitacao"
  | "dispensa" | "decreto" | "lei" | "portaria" | "resolucao"
  | "despacho" | "edital" | "ata" | "corrigenda" | "prestacao_contas" | "outro";

export interface ActHit {
  id: string;
  gazette_id: string;
  type: ActType;
  title: string;
  organ: string;
  organ_name: string;
  snippet: string;
  edition_number: string;
  published_at: string;
  is_extra: boolean;
  source_url: string;
  page_start: number | null;
  page_end: number | null;
  pdf_sha256: string;
  cnpjs: string[];
  values_cents: number[];
}

export interface SearchResponse {
  items: ActHit[];
  total: number;
  limit: number;
  offset: number;
}

export interface Organ {
  acronym: string;
  name: string;
  acts: number;
}

export interface CompanyResponse {
  cnpj: string;
  total_value_cents: number;
  count_by_type: Partial<Record<ActType, number>>;
  acts: ActHit[];
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body.error ?? `Erro ${res.status}`);
  return body as T;
}

export function searchActs(params: URLSearchParams) {
  return request<SearchResponse>(`/v1/acts?${params}`);
}

export function listOrgans() {
  return request<{ items: Organ[] }>("/v1/organs");
}

export function getCompany(cnpj: string) {
  return request<CompanyResponse>(`/v1/entities/cnpj/${encodeURIComponent(cnpj)}`);
}

export function subscribe(email: string, query: string) {
  return request<{ query: string; status: string }>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ email, query }),
  });
}

export function confirmSubscription(token: string) {
  return request<{ query: string }>("/v1/subscriptions/confirm", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

export function unsubscribe(token: string) {
  return request<unknown>("/v1/subscriptions/unsubscribe", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}
