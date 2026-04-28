export const MOBILE_KEYS = [
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
] as const;

export type ModifierState = { ctrl: boolean; alt: boolean };

export function applyMobileKey(key: string, state: ModifierState): { output: string; state: ModifierState } {
  if (key === "Ctrl") return { output: "", state: { ...state, ctrl: true } };
  if (key === "Alt") return { output: "", state: { ...state, alt: true } };

  let output = keyToSequence(key);
  if (state.ctrl && output.length === 1) {
    output = String.fromCharCode(output.toUpperCase().charCodeAt(0) - 64);
  } else if (key.startsWith("Ctrl+") && key.length === 6) {
    output = String.fromCharCode(key[5].charCodeAt(0) - 64);
  }
  return { output, state: { ctrl: false, alt: false } };
}

function keyToSequence(key: string): string {
  switch (key) {
    case "Esc":
      return "\u001b";
    case "Tab":
      return "\t";
    case "↑":
      return "\u001b[A";
    case "↓":
      return "\u001b[B";
    case "←":
      return "\u001b[D";
    case "→":
      return "\u001b[C";
    default:
      return key;
  }
}
