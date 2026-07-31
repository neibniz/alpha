import * as z from "zod";

export type GithubSchemaMessages = {
  clientIdRequired: string;
  clientSecretRequired: string;
  callbackUrlRequired: string;
  callbackUrlInvalid: string;
};

export function createGithubSchema(messages: GithubSchemaMessages) {
  return z
    .object({
      enabled: z.boolean(),
      client_id: z.string().optional(),
      client_secret: z.string().optional(),
      client_secret_configured: z.boolean(),
      redirect_url: z.string().optional(),
    })
    .superRefine((values, context) => {
      if (!values.enabled) {
        return;
      }
      if (!values.client_id?.trim()) {
        context.addIssue({
          code: "custom",
          message: messages.clientIdRequired,
          path: ["client_id"],
        });
      }
      if (!(values.client_secret_configured || values.client_secret?.trim())) {
        context.addIssue({
          code: "custom",
          message: messages.clientSecretRequired,
          path: ["client_secret"],
        });
      }

      const callback = values.redirect_url?.trim();
      if (!callback) {
        context.addIssue({
          code: "custom",
          message: messages.callbackUrlRequired,
          path: ["redirect_url"],
        });
        return;
      }
      try {
        const url = new URL(callback);
        if (
          !["http:", "https:"].includes(url.protocol) ||
          (url.pathname !== "" && url.pathname !== "/") ||
          url.search ||
          url.hash
        ) {
          throw new Error("invalid callback root");
        }
      } catch {
        context.addIssue({
          code: "custom",
          message: messages.callbackUrlInvalid,
          path: ["redirect_url"],
        });
      }
    });
}

export type GithubFormData = z.infer<ReturnType<typeof createGithubSchema>>;
