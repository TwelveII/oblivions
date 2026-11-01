/**
 * Auth API — authentication endpoints.
 */

import { apiClient } from "./client";
import type { AuthTokens, LoginRequest, RegisterRequest, User } from "../types";
import type { CredentialStoreFailure } from "../types/electron";

/** Register a new user. First user automatically becomes Owner. */
export async function register(data: RegisterRequest) {
  return apiClient<AuthTokens>("/auth/register", {
    method: "POST",
    body: data,
  });
}

export async function login(data: LoginRequest) {
  return apiClient<AuthTokens>("/auth/login", {
    method: "POST",
    body: data,
  });
}

export async function refreshToken(refresh_token: string) {
  return apiClient<{ access_token: string; refresh_token: string; file_token: string }>(
    "/auth/refresh",
    {
      method: "POST",
      body: { refresh_token },
    }
  );
}

export async function logout(refresh_token: string) {
  return apiClient<{ message: string }>("/auth/logout", {
    method: "POST",
    body: { refresh_token },
  });
}

export async function getMe() {
  return apiClient<User>("/users/me");
}

export type SignupInvite = {
  id: string;
  created_at: string;
  expires_at?: string;
  used_at?: string;
  revoked_at?: string;
};

export type CreatedSignupInvite = SignupInvite & { code: string };

export async function getSignupInvites() {
  return apiClient<SignupInvite[]>("/users/me/signup-invites");
}

export async function createSignupInvite(expiry_days: number) {
  return apiClient<CreatedSignupInvite>("/users/me/signup-invites", {
    method: "POST",
    body: { expiry_days },
  });
}

export async function revokeSignupInvite(inviteId: string) {
  return apiClient<{ message: string }>(`/users/me/signup-invites/${inviteId}`, {
    method: "DELETE",
  });
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
) {
  return apiClient<AuthTokens>("/users/me/password", {
    method: "POST",
    body: { current_password: currentPassword, new_password: newPassword },
  });
}

/** Sends a password reset email. Returns cooldown seconds if rate-limited. */


export async function dismissDownloadPrompt() {
  return apiClient<{ message: string }>("/users/me/dismiss-download-prompt", {
    method: "POST",
  });
}

/** Mark welcome modal as seen — never show again across devices. */
export async function dismissWelcome() {
  return apiClient<{ message: string }>("/users/me/dismiss-welcome", {
    method: "POST",
  });
}

/** Change email — requires current password for security. */

/** Permanently erases the account and reserves its username for 24 hours. */
export async function emergencyDeleteSelf(password: string) {
  return apiClient<{ message: string }>("/users/me/emergency", {
    method: "DELETE",
    body: { password },
  });
}

export async function softDeleteSelf(password: string) {
  return apiClient<{ message: string }>("/users/me", {
    method: "DELETE",
    body: { password },
  });
}

/**
 * Restores a soft-deleted account and returns auth tokens (immediate login).
 * Tombstones (admin-permanent-deleted) are not recoverable.
 */
export async function restoreAccount(username: string, password: string) {
  return apiClient<AuthTokens>("/auth/restore", {
    method: "POST",
    body: { username, password },
  });
}

/** Login error payload returned when account is soft-deleted. */
export interface AccountDeletedResponse {
  error: "account_deleted";
  username: string;
  deleted_at: string;
  permanent_delete_at: string;
}

/** Reports a "remember me" save or read that failed in the desktop app. */
export async function reportCredentialStoreFailure(operation: "save" | "load", failure: CredentialStoreFailure) {
  return apiClient<null>("/auth/credential-store-failure", {
    method: "POST",
    body: { operation, ...failure },
  });
}
