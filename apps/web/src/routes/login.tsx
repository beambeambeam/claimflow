import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useRouter, useSearch } from "@tanstack/react-router";
import { z } from "zod";

import { LoginForm } from "#/components/login-form";
import { getMeQueryKey } from "#/lib/api/@tanstack/react-query.gen";
import type { CurrentUserEnvelope } from "#/lib/api/types.gen";
import { getLandingPath } from "#/lib/landing";
import { isSafeRedirect } from "#/lib/redirect";

const LoginPage = () => {
  const { redirect } = useSearch({ from: "/login" });
  const queryClient = useQueryClient();
  const router = useRouter();

  const handleLoggedIn = (envelope: CurrentUserEnvelope) => {
    queryClient.setQueryData(getMeQueryKey(), envelope);

    const target = isSafeRedirect(redirect)
      ? redirect
      : getLandingPath(envelope.data.role);

    router.history.push(target);
  };

  return (
    <main className="flex flex-col items-start gap-6 p-8">
      <h1 className="text-2xl font-bold">Log in</h1>
      <LoginForm onLoggedIn={handleLoggedIn} />
    </main>
  );
};

export const Route = createFileRoute("/login")({
  validateSearch: z.object({ redirect: z.string().optional() }),
  component: LoginPage,
});
