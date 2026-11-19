import { expect, test } from "bun:test";
import { validateAuthFields } from "./auth-validation";

test("identifies individual fields rather than only a form-level error", () => {
  expect(
    validateAuthFields("login", { code: "", email: "", password: "" })
  ).toEqual({
    email: "请输入电子邮箱。",
    password: "请输入密码。",
  });
});

test("enforces reset requirements without applying them to login", () => {
  const values = { code: "", email: " user@example.test ", password: "short" };
  expect(validateAuthFields("reset", values)).toEqual({
    code: "请输入邮箱验证码。",
    password: "密码长度需要为 12 至 72 位。",
  });
  expect(validateAuthFields("login", values)).toEqual({});
});
