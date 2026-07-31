export type EmailAuthMode = "register" | "reset" | "login";

export type EmailAuthInitialValues = {
  email?: string;
  password?: string;
  repeat_password?: string;
  code?: string | null;
  invite?: string | null;
  cf_token?: string | null;
};

export type EmailLoginFormValues = {
  email: string;
  password: string;
  cf_token: string | undefined;
};

export type EmailRegisterFormValues = {
  email: string;
  password: string;
  repeat_password: string;
  code: string | null | undefined;
  invite: string | null | undefined;
  cf_token: string | null | undefined;
};

export type EmailResetFormValues = {
  email: string;
  password: string;
  code: string | null | undefined;
  cf_token: string | null | undefined;
};

export type EmailAuthFormValues =
  | EmailLoginFormValues
  | EmailRegisterFormValues
  | EmailResetFormValues;
