export type UserLifecycle = "enabled" | "disabled" | "deleted";

type UserDirectoryRow = Pick<
  API.UserListItem,
  | "balance"
  | "commission"
  | "deleted_at"
  | "enable"
  | "gift_amount"
  | "is_member"
>;

export function getUserLifecycle(
  user: Pick<API.UserListItem, "deleted_at" | "enable">
): UserLifecycle {
  if (user.deleted_at) return "deleted";
  return user.enable ? "enabled" : "disabled";
}

export function maskUserIdentifier(identifier: string, authType?: string) {
  const value = identifier.trim();
  if (!value) return "—";

  if (authType === "email" || value.includes("@")) {
    const [local = "", domain = ""] = value.split("@");
    if (!domain) return maskGenericIdentifier(value);
    const visible = local.slice(0, Math.min(2, local.length));
    return `${visible}${local.length > 2 ? "***" : "*"}@${domain}`;
  }

  if (
    authType === "mobile" ||
    authType === "phone" ||
    /^\+?[\d\s()-]{7,}$/.test(value)
  ) {
    const digits = value.replace(/\D/g, "");
    if (digits.length <= 7) return maskGenericIdentifier(value);
    return `${digits.slice(0, 3)}****${digits.slice(-4)}`;
  }

  return maskGenericIdentifier(value);
}

function maskGenericIdentifier(value: string) {
  if (value.length <= 2) return "*".repeat(value.length);
  if (value.length <= 6) return `${value.slice(0, 1)}***${value.slice(-1)}`;
  return `${value.slice(0, 2)}***${value.slice(-2)}`;
}

export function getUserWalletTotal(user: UserDirectoryRow) {
  return (
    Math.max(0, user.balance || 0) +
    Math.max(0, user.gift_amount || 0) +
    Math.max(0, user.commission || 0)
  );
}

export function buildUserPageSummary(
  users: UserDirectoryRow[],
  matchingTotal: number
) {
  let enabled = 0;
  let members = 0;
  let attention = 0;

  for (const user of users) {
    const lifecycle = getUserLifecycle(user);
    if (lifecycle === "enabled") enabled += 1;
    if (user.is_member && lifecycle !== "deleted") members += 1;
    if (lifecycle !== "enabled") attention += 1;
  }

  return {
    matchingTotal: Math.max(0, matchingTotal || 0),
    pageTotal: users.length,
    enabled,
    members,
    attention,
  };
}
