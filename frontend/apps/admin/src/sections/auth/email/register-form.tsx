import { AdminForm } from "@/components/admin-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@workspace/ui/components/alert";
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
import { Markdown } from "@workspace/ui/composed/markdown";
import { PasswordInput } from "@workspace/ui/composed/password-input";
import { isEmailDomainAllowed } from "@workspace/ui/utils/index";
import { CircleAlert, Mail, Ticket } from "lucide-react";
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
  EmailRegisterFormValues,
} from "./types";

export default function RegisterForm({
  loading,
  onSubmit,
  initialValues,
  setInitialValues,
  onSwitchForm,
}: {
  loading?: boolean;
  onSubmit: (data: EmailRegisterFormValues) => void;
  initialValues?: EmailAuthInitialValues;
  setInitialValues: Dispatch<
    SetStateAction<EmailAuthInitialValues | undefined>
  >;
  onSwitchForm: Dispatch<SetStateAction<EmailAuthMode>>;
}) {
  const { t } = useTranslation("auth");
  const { common } = useGlobalStore();
  const { verify, auth, invite } = common;

  const handleCheckUser = async (email: string) => {
    try {
      if (!auth.email.enable_domain_suffix) return true;
      return isEmailDomainAllowed(email, auth.email.domain_suffix_list);
    } catch (error) {
      console.log("Error checking user:", error);
      return false;
    }
  };

  const formSchema = z
    .object({
      email: z
        .string()
        .email(t("register.email", "Email"))
        .refine(handleCheckUser, {
          message: t("register.whitelist", "Email domain not allowed"),
        }),
      password: z.string(),
      repeat_password: z.string(),
      code: auth.email.enable_verify ? z.string() : z.string().nullish(),
      invite: invite.forced_invite ? z.string().min(1) : z.string().nullish(),
      cf_token:
        verify.enable_register_verify && verify.turnstile_site_key
          ? z.string()
          : z.string().nullish(),
    })
    .superRefine(({ password, repeat_password }, ctx) => {
      if (password !== repeat_password) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: t("register.passwordMismatch", "Passwords do not match"),
          path: ["repeat_password"],
        });
      }
    });

  const form = useForm<EmailRegisterFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: initialValues?.email ?? "",
      password: initialValues?.password ?? "",
      repeat_password: initialValues?.repeat_password ?? "",
      code: initialValues?.code ?? undefined,
      invite: localStorage.getItem("invite") || "",
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
      {auth.register.stop_register ? (
        <Alert>
          <CircleAlert />
          <AlertDescription>
            <Markdown>
              {t("register.message", "Registration is disabled")}
            </Markdown>
          </AlertDescription>
        </Alert>
      ) : (
        <Form {...form}>
          <AdminForm className="flex flex-col gap-5" onSubmit={handleSubmit}>
            <FieldGroup className="gap-5">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("register.emailLabel", "Email")}</FormLabel>
                    <InputGroup>
                      <InputGroupAddon>
                        <Mail />
                      </InputGroupAddon>
                      <FormControl>
                        <InputGroupInput
                          autoComplete="username"
                          placeholder={t(
                            "register.emailPlaceholder",
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
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("register.passwordLabel", "Password")}
                    </FormLabel>
                    <FormControl>
                      <PasswordInput
                        autoComplete="new-password"
                        placeholder={t(
                          "register.passwordPlaceholder",
                          "Enter your password..."
                        )}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="repeat_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("register.repeatPasswordLabel", "Confirm password")}
                    </FormLabel>
                    <FormControl>
                      <PasswordInput
                        autoComplete="new-password"
                        disabled={loading}
                        placeholder={t(
                          "register.repeatPasswordPlaceholder",
                          "Enter password again..."
                        )}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {auth.email.enable_verify && (
                <FormField
                  control={form.control}
                  name="code"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t("register.codeLabel", "Verification code")}
                      </FormLabel>
                      <InputGroup>
                        <FormControl>
                          <InputGroupInput
                            autoComplete="one-time-code"
                            disabled={loading}
                            placeholder={t(
                              "register.codePlaceholder",
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
                              type: 1,
                            }}
                          />
                        </InputGroupAddon>
                      </InputGroup>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
              <FormField
                control={form.control}
                name="invite"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("register.inviteLabel", "Invitation code")}
                    </FormLabel>
                    <InputGroup>
                      <InputGroupAddon>
                        <Ticket />
                      </InputGroupAddon>
                      <FormControl>
                        <InputGroupInput
                          disabled={loading || !!localStorage.getItem("invite")}
                          placeholder={t("register.invite", "Invite Code")}
                          {...field}
                          value={field.value || ""}
                        />
                      </FormControl>
                    </InputGroup>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {verify.enable_register_verify && (
                <FormField
                  control={form.control}
                  name="cf_token"
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <CloudFlareTurnstile
                          id="register"
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
            <Button
              className="w-full"
              disabled={loading}
              size="lg"
              type="submit"
            >
              {loading ? <Spinner /> : null}
              {t("register.title", "Register")}
            </Button>
          </AdminForm>
        </Form>
      )}
      <Separator className="my-4" />
      <div className="flex items-center justify-center gap-1 text-muted-foreground text-sm">
        <span>{t("register.existingAccount", "Already have an account?")}</span>
        <Button
          onClick={() => {
            setInitialValues(undefined);
            onSwitchForm("login");
          }}
          size="sm"
          variant="link"
        >
          {t("register.switchToLogin", "Login")}
        </Button>
      </div>
    </>
  );
}
