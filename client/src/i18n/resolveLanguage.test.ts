import { describe, it, expect } from "vitest";
import { resolveLanguage } from "./index";

describe("resolveLanguage", () => {
  it("always resolves to English", () => {
    for (const locale of ["tr", "tr-TR", "de", "fr-FR", "en-GB", "", undefined, null]) {
      expect(resolveLanguage(locale)).toBe("en");
    }
  });
});
