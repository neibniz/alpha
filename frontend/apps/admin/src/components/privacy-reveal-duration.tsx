import { Field, FieldLabel } from "@workspace/ui/components/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { useId } from "react";
import { useTranslation } from "react-i18next";
import {
  REVEAL_DURATION_MINUTES,
  type RevealDurationMinutes,
} from "@/lib/privacy-reveal-policy";

export function PrivacyRevealDuration({
  value,
  onChange,
}: {
  value: RevealDurationMinutes;
  onChange: (value: RevealDurationMinutes) => void;
}) {
  const id = useId();
  const { t } = useTranslation("log");
  return (
    <Field>
      <FieldLabel htmlFor={id}>
        {t("sensitive.duration", "查看时限")}
      </FieldLabel>
      <Select
        value={String(value)}
        onValueChange={(next) =>
          onChange(Number(next) as RevealDurationMinutes)
        }
      >
        <SelectTrigger id={id} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {REVEAL_DURATION_MINUTES.map((minutes) => (
              <SelectItem key={minutes} value={String(minutes)}>
                {t("sensitive.minutes", {
                  count: minutes,
                  defaultValue: "{{count}} 分钟",
                })}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  );
}
