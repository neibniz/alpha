"use client";

import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { Label } from "@workspace/ui/components/label";
import { Switch } from "@workspace/ui/components/switch";
import { DatePicker } from "@workspace/ui/composed/date-picker";
import { updateUserMembership } from "@workspace/ui/services/admin/user";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { formatDate } from "@/utils/common";

type UserWithMembership = API.User & {
  is_member?: boolean;
  member_expired_at?: number;
};

const defaultMembershipExpiry = () => {
  const date = new Date();
  date.setFullYear(date.getFullYear() + 1);
  return date.getTime();
};

export function MembershipInfoForm({
  user,
  refetch,
}: {
  user: UserWithMembership;
  refetch: () => Promise<unknown>;
}) {
  const { t } = useTranslation("user");
  const [isMember, setIsMember] = useState(Boolean(user.is_member));
  const [expiredAt, setExpiredAt] = useState(
    user.member_expired_at || defaultMembershipExpiry()
  );

  useEffect(() => {
    setIsMember(Boolean(user.is_member));
    setExpiredAt(user.member_expired_at || defaultMembershipExpiry());
  }, [user]);

  const handleMembershipChange = (checked: boolean) => {
    setIsMember(checked);
    if (checked && (!expiredAt || expiredAt <= Date.now())) {
      setExpiredAt(defaultMembershipExpiry());
    }
  };

  const handleSubmit = async () => {
    if (isMember && (!expiredAt || expiredAt <= Date.now())) {
      toast.error(
        t(
          "membershipExpireRequired",
          "Please choose a future membership expiry date"
        )
      );
      return;
    }
    await updateUserMembership({
      user_id: user.id,
      is_member: isMember,
      member_expired_at: isMember ? expiredAt : 0,
    });
    toast.success(t("updateSuccess", "Updated successfully"));
    await refetch();
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>{t("membership", "Membership")}</CardTitle>
          <CardDescription>
            {t(
              "membershipManageDescription",
              "Manage whether this user has a valid membership card and when it expires."
            )}
          </CardDescription>
        </div>
        <Badge variant={isMember ? "default" : "outline"}>
          {isMember
            ? t("memberActive", "Active Member")
            : t("memberInactive", "No Active Membership")}
        </Badge>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="flex items-center justify-between gap-4 rounded-lg border p-4">
          <div className="space-y-1">
            <Label>{t("membershipValid", "Membership Valid")}</Label>
            <p className="text-muted-foreground text-sm">
              {t(
                "membershipValidDescription",
                "Disabling it clears the user's membership expiry time."
              )}
            </p>
          </div>
          <Switch checked={isMember} onCheckedChange={handleMembershipChange} />
        </div>

        <div className="grid gap-2">
          <Label>{t("membershipExpiresAt", "Membership Expires At")}</Label>
          <div className={isMember ? "" : "pointer-events-none opacity-60"}>
            <DatePicker
              key={`${isMember}-${expiredAt}`}
              onChange={(value) =>
                setExpiredAt(value || defaultMembershipExpiry())
              }
              placeholder={t("selectDate", "Select date")}
              value={isMember ? expiredAt : 0}
            />
          </div>
          <p className="text-muted-foreground text-xs">
            {isMember
              ? `${t("currentExpiry", "Current expiry")}: ${formatDate(
                  expiredAt,
                  false
                )}`
              : t(
                  "membershipDisabledHint",
                  "The user will not be treated as a member after saving."
                )}
          </p>
        </div>

        <div className="flex justify-end">
          <Button onClick={handleSubmit} type="button">
            {t("save", "Save")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
