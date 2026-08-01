const GITHUB_CALLBACK_PATHS = {
  bind: "/bind/github/callback",
  login: "/oauth/github/callback",
} as const;

export type GitHubOAuthFlow = keyof typeof GITHUB_CALLBACK_PATHS;

export function githubOAuthCallbackUrl(origin: string, flow: GitHubOAuthFlow) {
  const callback = new URL(GITHUB_CALLBACK_PATHS[flow], origin);
  return callback.toString();
}
