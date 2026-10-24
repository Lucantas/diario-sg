import { readFileSync, writeFileSync } from "node:fs";

const FILE = new URL("../src/doacoes.json", import.meta.url);
const USAGE = "uso: make doacao TOTAL=85,50 (total recebido no mês até hoje, em reais)";

function cents(text) {
  const raw = String(text ?? "").trim();
  const normalized = raw.includes(",") ? raw.replace(/\./g, "").replace(",", ".") : raw;
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) throw new Error(USAGE);
  return Math.round(Number(normalized) * 100);
}

function today() {
  const now = new Date();
  const pad = (n) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

try {
  const current = JSON.parse(readFileSync(FILE, "utf8"));
  const date = today();
  const next = { ...current, month: date.slice(0, 7), received_cents: cents(process.argv[2]), updated_on: date };
  writeFileSync(FILE, `${JSON.stringify(next, null, 2)}\n`);
  const percent = Math.floor((next.received_cents / next.goal_cents) * 100);
  const brl = (c) => (c / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
  console.log(`${next.month}: ${brl(next.received_cents)} de ${brl(next.goal_cents)} (${percent}%)`);
} catch (err) {
  console.error(err.message);
  process.exit(1);
}
