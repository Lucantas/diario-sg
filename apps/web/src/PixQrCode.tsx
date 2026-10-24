import { useMemo } from "react";
import QRCode from "qrcode";

const QUIET_ZONE = 4;

function modulesPath(payload: string): { path: string; size: number } {
  const { modules } = QRCode.create(payload, { errorCorrectionLevel: "M" });
  const steps: string[] = [];
  for (let row = 0; row < modules.size; row++) {
    for (let col = 0; col < modules.size; col++) {
      if (modules.get(row, col)) steps.push(`M${col + QUIET_ZONE} ${row + QUIET_ZONE}h1v1h-1z`);
    }
  }
  return { path: steps.join(""), size: modules.size + 2 * QUIET_ZONE };
}

export function PixQrCode({ payload }: { payload: string }) {
  const { path, size } = useMemo(() => modulesPath(payload), [payload]);
  return (
    <svg className="pix-qr" viewBox={`0 0 ${size} ${size}`} role="img" aria-label="QR Code do PIX" shapeRendering="crispEdges">
      <rect width={size} height={size} className="pix-qr-bg" />
      <path d={path} className="pix-qr-ink" />
    </svg>
  );
}
