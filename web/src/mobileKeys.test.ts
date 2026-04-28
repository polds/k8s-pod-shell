import { describe, expect, it } from "vitest";
import { applyMobileKey, MOBILE_KEYS } from "./mobileKeys";

describe("mobile key ordering", () => {
  it("keeps exact required key order", () => {
    expect(MOBILE_KEYS).toEqual([
      "Esc",
      "Tab",
      "Ctrl",
      "Alt",
      "↑",
      "↓",
      "←",
      "→",
      "Ctrl+C",
      "Ctrl+D",
      "Ctrl+Z",
      "Ctrl+L",
      "Ctrl+A",
      "Ctrl+E",
      "Ctrl+U",
      "Ctrl+K",
      "Ctrl+R"
    ]);
  });
});

describe("modifier behavior", () => {
  it("supports sticky one-shot ctrl", () => {
    const armed = applyMobileKey("Ctrl", { ctrl: false, alt: false });
    expect(armed.state.ctrl).toBe(true);
    const fired = applyMobileKey("c", armed.state);
    expect(fired.output.charCodeAt(0)).toBe(3);
    expect(fired.state.ctrl).toBe(false);
  });
});
