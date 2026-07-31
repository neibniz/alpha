import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@workspace/ui/components/button";
import { FieldGroup } from "@workspace/ui/components/field";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@workspace/ui/components/form";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import { Separator } from "@workspace/ui/components/separator";
import { Spinner } from "@workspace/ui/components/spinner";
import { PasswordInput } from "@workspace/ui/composed/password-input";
import { ArrowLeft, Mail } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { useRef } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { useGlobalStore } from "@/stores/global";
import SendCode from "../send-code";
import CloudFlareTurnstile, { type TurnstileRef } from "../turnstile";
import type {
  EmailAuthInitialValues,
  EmailAuthMode,
  EmailResetFormValues,
} from "./types";

export default function ResetForm({
  loading,
  onSubmit,
  initialValues,
  setInitialValues,
  onSwitchForm,
}: {
  loading?: boolean;
  onSubmit: (data: EmailResetFormValues) => void;
  initialValues?: EmailAuthInitialValues;
  setInitialValues: Dispatch<
    SetStateAction<EmailAuthInitialValues | undefined>
  >;
  onSwitchForm: Dispatch<SetStateAction<EmailAuthMode>>;
}) {
  const { t } = useTranslation("auth");

  const { common } = useGlobalStore();
  const { verify, auth } = common;

  const formSchema = z.object({
    email: z.email(t("reset.email", "Email")),
    password: z.string(),
    code: auth?.email?.enable_verify ? z.string() : z.string().nullish(),
    cf_token:
      verify.enable_register_verify && verify.turnstile_site_key
        ? z.string()
        : z.string().nullish(),
  });
  const form = useForm<EmailResetFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: initialValues?.email ?? "",
      password: initialValues?.password ?? "",
      code: initialValues?.code ?? undefined,
      cf_token: initialValues?.cf_token ?? undefined,
    },
  });

  const turnstile = useRef<TurnstileRef>(null);
  const handleSubmit = form.handleSubmit((data) => {
    try {
      onSubmit(data);
    } catch (_error) {
      turnstile.current?.reset();
    }
  });

  return (
    <>
      <Form {...form}>
        <form className="flex flex-col gap-5" onSubmit={handleSubmit}>
          <FieldGroup className="gap-5">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("reset.emailLabel", "Email")}</FormLabel>
                  <InputGroup>
                    <InputGroupAddon>
                      <Mail />
                    </InputGroupAddon>
                    <FormControl>
                      <InputGroupInput
                        autoComplete="username"
                        placeholder={t(
                          "reset.emailPlaceholder",
                          "Enter your email..."
                        )}
                        type="email"
                        {...field}
                      />
                    </FormControl>
                  </InputGroup>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="code"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t("reset.codeLabel", "Verification code")}
                  </FormLabel>
                  <InputGroup>
                    <FormControl>
                      <InputGroupInput
                        autoComplete="one-time-code"
                        disabled={loading}
                        placeholder={t(
                          "reset.codePlaceholder",
                          "Enter code..."
                        )}
                        type="text"
                        {...field}
                        value={field.value as string}
                      />
                    </FormControl>
                    <InputGroupAddon align="inline-end">
                      <SendCode
                        params={{
                          email: form.watch("email"),
                          type: 2,
                        }}
                      />
                    </InputGroupAddon>
                  </InputGroup>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t("reset.passwordLabel", "New password")}
                  </FormLabel>
                  <FormControl>
                    <PasswordInput
                      autoComplete="new-password"
                      placeholder={t(
                        "reset.passwordPlaceholder",
                        "Enter your new password..."
                      )}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {verify.enable_reset_password_verify && (
              <FormField
                control={form.control}
                name="cf_token"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CloudFlareTurnstile
                        id="reset"
                        {...field}
                        ref={turnstile}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </FieldGroup>
          <Button className="w-full" disabled={loading} size="lg" type="submit">
            {loading ? <Spinner /> : null}
            {t("reset.title", "Reset Password")}
          </Button>
        </form>
      </Form>
      <Separator className="my-4" />
      <div className="flex items-center justify-center gap-1 text-muted-foreground text-sm">
        <span>{t("reset.existingAccount", "Remember your password?")}</span>
        <Button
          onClick={() => {
            setInitialValues(undefined);
            onSwitchForm("login");
          }}
          size="sm"
          variant="link"
        >
          <ArrowLeft data-icon="inline-start" />
          {t("reset.switchToLogin", "Login")}
        </Button>
      </div>
    </>
  );
}
