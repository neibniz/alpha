import { useTheme } from "next-themes";
import { type RefObject, useEffect, useImperativeHandle } from "react";
import { useTranslation } from "react-i18next";
import Turnstile, { useTurnstile } from "react-turnstile";

import { useGlobalStore } from "@/stores/global";

export type TurnstileRef = {
  reset: () => void;
};

const CloudFlareTurnstile = function CloudFlareTurnstile({
  id,
  value,
  onChange,
  ref,
}: {
  id?: string;
  value?: null | string;
  onChange: (value?: string) => void;
  ref?: RefObject<TurnstileRef | null>;
}) {
  const { common } = useGlobalStore();
  const { verify } = common;
  const { resolvedTheme } = useTheme();
  const { i18n } = useTranslation();
  const locale = i18n.language;
  const turnstile = useTurnstile();

  useImperativeHandle(
    ref,
    () => ({
      reset: () => turnstile.reset(),
    }),
    [turnstile]
  );

  useEffect(() => {
    if (value === "") {
      turnstile.reset();
    }
  }, [turnstile, value]);

  return (
    verify.turnstile_site_key && (
      <Turnstile
        fixedSize
        id={id}
        language={locale.toLowerCase()}
        onExpire={() => {
          onChange();
          turnstile.reset();
        }}
        onTimeout={() => {
          onChange();
          turnstile.reset();
        }}
        onVerify={(token) => onChange(token)}
        sitekey={verify.turnstile_site_key}
        theme={resolvedTheme as "light" | "dark"}
      />
    )
  );
};

export default CloudFlareTurnstile;
