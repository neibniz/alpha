import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@workspace/ui/components/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@workspace/ui/components/form";
import { FieldGroup } from "@workspace/ui/components/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import { Separator } from "@workspace/ui/components/separator";
import { Spinner } from "@workspace/ui/components/spinner";
import { PasswordInput } from "@workspace/ui/composed/password-input";
import { KeyRound, Mail } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { useRef } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { useGlobalStore } from "@/stores/global";
import CloudFlareTurnstile, { type TurnstileRef } from "../turnstile";
import type {
  EmailAuthInitialValues,
  EmailAuthMode,
  EmailLoginFormValues,
} from "./types";

export default function LoginForm({
  loading,
  onSubmit,
  initialValues,
  // setInitialValues,
  onSwitchForm,
}: {
  loading?: boolean;
  onSubmit: (data: EmailLoginFormValues) => void;
  initialValues?: EmailAuthInitialValues;
  setInitialValues: Dispatch<
    SetStateAction<EmailAuthInitialValues | undefined>
  >;
  onSwitchForm: Dispatch<SetStateAction<EmailAuthMode>>;
}) {
  const { t } = useTranslation("auth");
  const { common } = useGlobalStore();
  const { verify } = common;

  const formSchema = z.object({
    email: z
      .string()
      .trim()
      .min(1, t("login.emailRequired", "Email is required")),
    password: z.string(),
    cf_token:
      verify.enable_login_verify && verify.turnstile_site_key
        ? z.string()
        : z.string().optional(),
  });
  const form = useForm<EmailLoginFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: initialValues?.email ?? "",
      password: initialValues?.password ?? "",
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
                  <FormLabel>{t("login.emailLabel", "Email")}</FormLabel>
                  <InputGroup>
                    <InputGroupAddon>
                      <Mail />
                    </InputGroupAddon>
                    <FormControl>
                      <InputGroupInput
                        autoComplete="username"
                        placeholder={t(
                          "login.emailPlaceholder",
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
                  <FormLabel>{t("login.passwordLabel", "Password")}</FormLabel>
                  <FormControl>
                    <PasswordInput
                      autoComplete="current-password"
                      placeholder={t(
                        "login.passwordPlaceholder",
                        "Enter your password..."
                      )}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {verify.enable_login_verify && (
              <FormField
                control={form.control}
                name="cf_token"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CloudFlareTurnstile
                        id="login"
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
            {t("login.title", "Login")}
          </Button>
        </form>
      </Form>
      <Separator className="my-4" />
      <div className="flex justify-center">
        <Button
          onClick={() => onSwitchForm("reset")}
          size="sm"
          type="button"
          variant="link"
        >
          <KeyRound data-icon="inline-start" />
          {t("login.forgotPassword", "Forgot Password?")}
        </Button>
      </div>
    </>
  );
}
