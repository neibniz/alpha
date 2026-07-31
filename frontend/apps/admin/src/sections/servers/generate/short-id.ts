/**
 * Generate a short ID for Reality
 * @returns A random hexadecimal string of length 2, 4, 6, 8, 10, 12, 14, or 16
 */
export function generateRealityShortId() {
  const hex = "0123456789abcdef";
  const lengths = [2, 4, 6, 8, 10, 12, 14, 16];
  const idx = Math.floor(Math.random() * lengths.length);
  const len = lengths[idx] ?? 16;
  let out = "";
  for (let i = 0; i < len; i++) {
    out += hex.charAt(Math.floor(Math.random() * hex.length));
  }
  return out;
}
