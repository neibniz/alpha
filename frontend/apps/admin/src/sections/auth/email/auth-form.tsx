"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import {
  resetPassword,
  userLogin,
  userRegister,
} from "@workspace/ui/services/common/auth";
import type { Dispatch, SetStateAction } from "react";
import { useState, useTransition } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { USER_EMAIL, USER_PASSWORD } from "@/config";
import { refreshAdminSession } from "@/lib/admin-bootstrap";
import { getRedirectUrl, setAuthorization } from "@/utils/common";
import LoginForm from "./login-form";
import RegisterForm from "./register-form";
import ResetForm from "./reset-form";
import type {
  EmailAuthFormValues,
  EmailAuthInitialValues,
  EmailAuthMode,
  EmailLoginFormValues,
  EmailRegisterFormValues,
  EmailResetFormValues,
} from "./types";

export default function EmailAuthForm({
  mode,
  onModeChange,
}: {
  mode: EmailAuthMode;
  onModeChange: Dispatch<SetStateAction<EmailAuthMode>>;
}) {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const type = mode;
  const setType = onModeChange;
  const [loading, startTransition] = useTransition();
  const [initialValues, setInitialValues] = useState<
    EmailAuthInitialValues | undefined
  >({
    email: USER_EMAIL,
    password: USER_PASSWORD,
  });

  const handleFormSubmit = async (params: EmailAuthFormValues) => {
    const onLogin = async (token?: string) => {
      if (!token) return;
      setAuthorization(token);
      await refreshAdminSession(queryClient);
      navigate({ to: getRedirectUrl() });
    };
    startTransition(async () => {
      try {
        switch (type) {
          case "login": {
            const values = params as EmailLoginFormValues;
            const login = await userLogin({
              identifier: "",
              email: values.email,
              password: values.password,
              cf_token: values.cf_token,
            });
            toast.success(t("login.success", "Login successful!"));
            await onLogin(login.data.data?.token);
            break;
          }
          case "register": {
            const values = params as EmailRegisterFormValues;
            const create = await userRegister({
              identifier: "",
              email: values.email,
              password: values.password,
              invite: values.invite ?? undefined,
              code: values.code ?? undefined,
              cf_token: values.cf_token ?? undefined,
            });
            toast.success(t("register.success", "Registration successful!"));
            await onLogin(create.data.data?.token);
            break;
          }
          case "reset": {
            const values = params as EmailResetFormValues;
            await resetPassword({
              identifier: "",
              email: values.email,
              password: values.password,
              code: values.code ?? undefined,
              cf_token: values.cf_token ?? undefined,
            });
            toast.success(t("reset.success", "Password reset successful!"));
            setType("login");
            break;
          }
        }
      } catch (_error) {
        /* empty */
      }
    });
  };

  switch (type) {
    case "login":
      return (
        <LoginForm
          initialValues={initialValues}
          loading={loading}
          onSubmit={handleFormSubmit}
          onSwitchForm={setType}
          setInitialValues={setInitialValues}
        />
      );
    case "register":
      return (
        <RegisterForm
          initialValues={initialValues}
          loading={loading}
          onSubmit={handleFormSubmit}
          onSwitchForm={setType}
          setInitialValues={setInitialValues}
        />
      );
    case "reset":
      return (
        <ResetForm
          initialValues={initialValues}
          loading={loading}
          onSubmit={handleFormSubmit}
          onSwitchForm={setType}
          setInitialValues={setInitialValues}
        />
      );
    default:
      return null;
  }
}
