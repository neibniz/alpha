import { uid } from "radash";

/**
 * Generate a random password
 * @param length Length of the password
 * @param charset Character set to use (defaults to alphanumeric)
 * @returns Randomly generated password
 */
export function generatePassword(length = 16, charset?: string) {
  return uid(length, charset).toLowerCase();
}
