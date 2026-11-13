import { AppShell } from "@astryxdesign/core/AppShell";
import { Card } from "@astryxdesign/core/Card";
import { Center } from "@astryxdesign/core/Center";
import { VStack } from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";
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
    <AppShell contentPadding={6}>
      <Center axis="both" height="100%">
        <Card maxWidth={400} padding={8} width="100%">
          <VStack gap={4} hAlign="stretch">
            <VStack gap={1}>
              <Text type="body" color="secondary" weight="bold">
                ClaimFlow
              </Text>
              <Heading level={1}>Log in</Heading>
            </VStack>
            <LoginForm onLoggedIn={handleLoggedIn} />
          </VStack>
        </Card>
      </Center>
    </AppShell>
  );
};

export const Route = createFileRoute("/login")({
  validateSearch: z.object({ redirect: z.string().optional() }),
  component: LoginPage,
});
