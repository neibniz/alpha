export type AuthMode = "login" | "register" | "reset";
export type AuthErrors = Partial<Record<"email" | "code" | "password", string>>;

export function validateAuthEmail(value: string): string | undefined {
  const email = value.trim();
  if (!email) return "请输入电子邮箱。";
  if (!/^[^\s@]+@[^\s@]+$/u.test(email)) return "请输入有效的电子邮箱。";
}

export function validateAuthFields(
  mode: AuthMode,
  values: { email: string; code: string; password: string }
): AuthErrors {
  const errors: AuthErrors = {};
  const emailError = validateAuthEmail(values.email);
  if (emailError) errors.email = emailError;
  if (mode === "reset" && !values.code) {
    errors.code = "请输入邮箱验证码。";
  }
  if (!values.password) {
    errors.password = "请输入密码。";
  } else if (mode === "login" && values.password.length > 128) {
    errors.password = "密码不能超过 128 个字符。";
  } else if (
    mode !== "login" &&
    (values.password.length < 12 || values.password.length > 72)
  ) {
    errors.password = "密码长度需要为 12 至 72 位。";
  }
  return errors;
}
