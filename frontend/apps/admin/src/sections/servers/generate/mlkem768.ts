import mlkem from "mlkem-wasm";
import { toB64Url } from "./util";

export async function generateMLKEM768KeyPair() {
  const mlkemKeyPair = await mlkem.generateKey({ name: "ML-KEM-768" }, true, [
    "encapsulateBits",
    "decapsulateBits",
  ]);
  const mlkemPublicKeyRaw = await mlkem.exportKey(
    "raw-public",
    mlkemKeyPair.publicKey
  );
  const mlkemPrivateKeyRaw = await mlkem.exportKey(
    "raw-seed",
    mlkemKeyPair.privateKey
  );

  return {
    publicKey: toB64Url(new Uint8Array(mlkemPublicKeyRaw)),
    privateKey: toB64Url(new Uint8Array(mlkemPrivateKeyRaw)),
  };
}
