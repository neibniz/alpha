import { x25519 } from "@noble/curves/ed25519.js";
import { toB64Url } from "./util";

/**
 * Generate a Reality key pair
 * @returns An object containing the private and public keys in base64url format
 */
export function generateRealityKeyPair() {
  const { secretKey, publicKey } = x25519.keygen();
  return { privateKey: toB64Url(secretKey), publicKey: toB64Url(publicKey) };
}
