import { describe, expect, it } from "vitest";
import { formatBytes } from "./format";

describe("formatBytes", () => {
  it("usa a unidade mais próxima com vírgula decimal", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(900)).toBe("900 B");
    expect(formatBytes(1536)).toBe("1,5 KB");
    expect(formatBytes(54_085_626)).toBe("51,6 MB");
    expect(formatBytes(3 * 1024 ** 3)).toBe("3,0 GB");
  });
});
