import { pixPayload } from "./pix";

export const PIX_RECEIVER_NAME = "Lucas Dantas de Oliveira";
const PIX_RECEIVER_CITY = "São Gonçalo";

export interface DonationPix {
  key: string;
  payload: string;
}

export function donationPix(rawKey: string = import.meta.env.VITE_PIX_KEY ?? ""): DonationPix | null {
  const key = rawKey.trim();
  if (!key) return null;
  return { key, payload: pixPayload({ key, name: PIX_RECEIVER_NAME, city: PIX_RECEIVER_CITY }) };
}
