const DEFAULT_EMAIL_AUTH_METHOD: API.UserAuthMethod = {
  auth_type: "email",
  auth_identifier: "",
  verified: false,
};

const VISIBLE_AUTH_METHODS = new Set(["email", "github"]);

export function buildUserAuthMethodData(
  methods: readonly API.UserAuthMethod[] | undefined
) {
  let emailMethod: API.UserAuthMethod | undefined;
  const otherMethods: API.UserAuthMethod[] = [];
  if (methods?.length) {
    for (const method of methods) {
      if (!VISIBLE_AUTH_METHODS.has(method.auth_type)) {
        continue;
      }
      if (method.auth_type === "email") {
        if (!emailMethod) {
          emailMethod = method;
        }
        continue;
      }
      otherMethods.push(method);
    }
  }
  return {
    emailMethod,
    defaultEmailMethod: {
      ...DEFAULT_EMAIL_AUTH_METHOD,
      ...emailMethod,
    },
    isEmailExists: Boolean(emailMethod),
    otherMethods,
  };
}

export function preferredAuthIdentifier(
  methods: readonly API.UserAuthMethod[] | undefined
) {
  if (!methods?.length) {
    return "";
  }
  const firstIdentifier = methods[0]?.auth_identifier || "";
  for (const method of methods) {
    if (method.auth_type === "email" && method.auth_identifier) {
      return method.auth_identifier;
    }
  }
  return firstIdentifier;
}
