export interface PixReceiver {
  key: string;
  name: string;
  city: string;
}

const MAX_NAME = 25;
const MAX_CITY = 15;

function field(id: string, value: string): string {
  return `${id}${String(value.length).padStart(2, "0")}${value}`;
}

function plain(text: string, max: number): string {
  return text.normalize("NFD").replace(/[̀-ͯ]/g, "").slice(0, max);
}

function crc16(data: string): string {
  let crc = 0xffff;
  for (const char of data) {
    crc ^= char.charCodeAt(0) << 8;
    for (let bit = 0; bit < 8; bit++) {
      crc = crc & 0x8000 ? ((crc << 1) ^ 0x1021) & 0xffff : (crc << 1) & 0xffff;
    }
  }
  return crc.toString(16).toUpperCase().padStart(4, "0");
}

export function pixPayload({ key, name, city }: PixReceiver): string {
  const body = [
    field("00", "01"),
    field("26", field("00", "br.gov.bcb.pix") + field("01", key)),
    field("52", "0000"),
    field("53", "986"),
    field("58", "BR"),
    field("59", plain(name, MAX_NAME)),
    field("60", plain(city, MAX_CITY)),
    field("62", field("05", "***")),
  ].join("");
  const withCrcHeader = `${body}6304`;
  return withCrcHeader + crc16(withCrcHeader);
}
