import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { ConfirmButton } from "@/components/confirm-button";
import {
  createUserAuthMethod,
  deleteUserAuthMethod,
  updateUserAuthMethod,
} from "@workspace/ui/services/admin/user";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { buildUserAuthMethodData } from "../user-auth-method-data";

export function AuthMethodsForm({
  user,
  refetch,
}: {
  user: API.User;
  refetch: () => Promise<unknown>;
}) {
  const { t } = useTranslation("user");

  const [emailChanges, setEmailChanges] = useState<Record<string, string>>({});
  const [savingEmail, setSavingEmail] = useState(false);
  const [removingType, setRemovingType] = useState<string>();

  const handleRemoveAuth = async (authType: string) => {
    setRemovingType(authType);
    try {
      await deleteUserAuthMethod({
        user_id: user.id,
        auth_type: authType,
      });
      toast.success(t("deleteSuccess", "Deleted successfully"));
      await refetch();
    } finally {
      setRemovingType(undefined);
    }
  };

  const handleUpdateEmail = async (email: string) => {
    setSavingEmail(true);
    try {
      await updateUserAuthMethod({
        user_id: user.id,
        auth_type: "email",
        auth_identifier: email,
      });
      toast.success(t("updateSuccess", "Updated successfully"));
      setEmailChanges({});
      await refetch();
    } finally {
      setSavingEmail(false);
    }
  };

  const handleCreateEmail = async (email: string) => {
    setSavingEmail(true);
    try {
      await createUserAuthMethod({
        user_id: user.id,
        auth_type: "email",
        auth_identifier: email,
      });
      toast.success(t("createSuccess", "Created successfully"));
      setEmailChanges({});
      await refetch();
    } finally {
      setSavingEmail(false);
    }
  };

  const handleEmailChange = (authType: string, value: string) => {
    setEmailChanges((prev) => ({
      ...prev,
      [authType]: value,
    }));
  };

  const { defaultEmailMethod, isEmailExists, otherMethods } =
    buildUserAuthMethodData(user.auth_methods);
  const emailDraft = emailChanges.email ?? defaultEmailMethod.auth_identifier;
  const normalizedEmail = emailDraft.trim().toLowerCase();
  const emailChanged =
    emailChanges.email !== undefined &&
    normalizedEmail !== defaultEmailMethod.auth_identifier.trim().toLowerCase();
  const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail);
  const canRemoveAuthMethod = user.auth_methods.length > 1;

  const handleEmailAction = async () => {
    if (isEmailExists) {
      await handleUpdateEmail(normalizedEmail);
    } else {
      await handleCreateEmail(normalizedEmail);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("authMethodsTitle", "Auth Methods")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="divide-y">
          <section className="space-y-3 pb-5">
            <div className="flex items-center justify-between">
              <div className="font-medium uppercase">email</div>
              <Badge
                variant={
                  defaultEmailMethod.verified ? "default" : "destructive"
                }
              >
                {defaultEmailMethod.verified
                  ? t("verified", "Verified")
                  : t("unverified", "Unverified")}
              </Badge>
            </div>
            <div className="flex items-center gap-2">
              <div className="flex-1">
                <EnhancedInput
                  disabled={savingEmail}
                  onValueChange={(value) =>
                    handleEmailChange("email", value as string)
                  }
                  placeholder={t("pleaseEnterEmail", "Enter email")}
                  type="email"
                  value={emailDraft}
                />
              </div>
              <Button
                disabled={
                  savingEmail || !emailValid || (isEmailExists && !emailChanged)
                }
                onClick={handleEmailAction}
              >
                {savingEmail
                  ? t("saving", "Saving")
                  : isEmailExists
                    ? t("update", "Update")
                    : t("add", "Add")}
              </Button>
            </div>
          </section>

          {otherMethods.map((method) => (
            <section className="space-y-3 py-5" key={method.auth_type}>
              <div className="flex items-center justify-between">
                <div className="font-medium uppercase">{method.auth_type}</div>
                <Badge variant={method.verified ? "default" : "destructive"}>
                  {method.verified
                    ? t("verified", "Verified")
                    : t("unverified", "Unverified")}
                </Badge>
              </div>
              <div className="flex items-center gap-4">
                <div className="flex-1">
                  <div className="text-muted-foreground text-sm">
                    {method.auth_identifier}
                  </div>
                </div>
                <ConfirmButton
                  cancelText={t("cancel", "Cancel")}
                  confirmText={t("remove", "Remove")}
                  description={t(
                    "removeAuthMethodDescription",
                    "This sign-in method will no longer grant access to the account."
                  )}
                  onConfirm={() => handleRemoveAuth(method.auth_type)}
                  title={t("removeAuthMethodTitle", "Remove sign-in method?")}
                  trigger={
                    <Button
                      disabled={
                        !canRemoveAuthMethod ||
                        removingType === method.auth_type
                      }
                      size="sm"
                      title={
                        canRemoveAuthMethod
                          ? undefined
                          : t(
                              "keepOneAuthMethod",
                              "At least one sign-in method is required."
                            )
                      }
                      variant="destructive"
                    >
                      {removingType === method.auth_type
                        ? t("removing", "Removing")
                        : t("remove", "Remove")}
                    </Button>
                  }
                />
              </div>
            </section>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
