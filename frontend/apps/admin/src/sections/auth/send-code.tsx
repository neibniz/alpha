import { InputGroupButton } from "@workspace/ui/components/input-group";
import { sendEmailCode } from "@workspace/ui/services/common/common";
import { useCountDown } from "ahooks";
import { useState } from "react";
import { useTranslation } from "react-i18next";

interface SendCodeProps {
  params: {
    email: string;
    type?: 1 | 2;
  };
}
export default function SendCode({ params }: SendCodeProps) {
  const { t } = useTranslation("auth");
  const [targetDate, setTargetDate] = useState<number>();

  const [, { seconds }] = useCountDown({
    targetDate,
    onEnd: () => {
      setTargetDate(undefined);
    },
  });

  const getEmailCode = async () => {
    if (params.email && params.type) {
      await sendEmailCode({
        email: params.email,
        type: params.type,
      });
      setTargetDate(Date.now() + 60_000);
    }
  };

  const handleSendCode = async () => {
    await getEmailCode();
  };
  const disabled = seconds > 0 || !params.email;

  return (
    <InputGroupButton disabled={disabled} onClick={handleSendCode} size="sm">
      {seconds > 0 ? `${seconds}s` : t("get", "Get Code")}
    </InputGroupButton>
  );
}
