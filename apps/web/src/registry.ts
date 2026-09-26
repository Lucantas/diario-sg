export interface AddressParts {
  street: string;
  number: string;
  complement: string;
  district: string;
  city: string;
  uf: string;
  zip: string;
}

const MONTHS = ["janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"];
const ZIP_DIGITS = 8;
const ZIP_PREFIX = 5;

export function formatAddress(a: AddressParts): string {
  const zip = a.zip.length === ZIP_DIGITS ? `CEP ${a.zip.slice(0, ZIP_PREFIX)}-${a.zip.slice(ZIP_PREFIX)}` : "";
  const city = [a.city, a.uf].filter(Boolean).join("/");
  return [a.street, a.number, a.complement, a.district, city, zip].filter(Boolean).join(", ");
}

export function isActive(status: string): boolean {
  return status === "Ativa";
}

export function formatIsoDate(iso: string | null): string {
  if (!iso) return "";
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
}

export function formatRegistryMonth(month: string): string {
  const [y, m] = month.split("-");
  return `${MONTHS[Number(m) - 1]} de ${y}`;
}
