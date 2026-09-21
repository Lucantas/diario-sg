// Cliente da API. Tudo passa por /api: em produção o nginx faz o proxy para
// o serviço de backend; em dev, o Vite faz o mesmo.

export type ActType =
  | "nomeacao" | "exoneracao" | "contrato" | "aditivo" | "licitacao"
  | "dispensa" | "decreto" | "lei" | "portaria" | "outro";

export interface ActHit {
  id: string;
  gazette_id: string;
  type: ActType;
  title: string;
  snippet: string;
  edition_number: string;
  published_at: string;
}

export interface SearchResponse {
  items: ActHit[];
  total: number;
  limit: number;
  offset: number;
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

export function searchActs(q: string, type: ActType | "", offset = 0) {
  const params = new URLSearchParams({ q, offset: String(offset) });
  if (type) params.set("type", type);
  return request<SearchResponse>(`/v1/acts?${params}`);
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
