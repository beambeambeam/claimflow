import { Button } from "@astryxdesign/core/Button";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Text } from "@astryxdesign/core/Text";
import { TopNav, TopNavItem } from "@astryxdesign/core/TopNav";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter, useRouterState } from "@tanstack/react-router";

import { can, roleLabels } from "#/features/auth/permissions";
import { logoutMutation } from "#/lib/api/@tanstack/react-query.gen";
import type { CurrentUser } from "#/lib/api/types.gen";

export const AppHeader = ({ user }: { user: CurrentUser }) => {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });

  const queryClient = useQueryClient();
  const router = useRouter();

  // Whatever the server answers, this browser no longer has a session: drop cached data and show login.
  const logout = useMutation({
    ...logoutMutation(),
    onSettled: async () => {
      queryClient.clear();
      await router.navigate({ to: "/login" });
    },
  });

  return (
    <TopNav
      endContent={
        <HStack gap={4} vAlign="center">
          <VStack gap={0} hAlign="end">
            <Text type="body" weight="bold">
              {user.name}
            </Text>
            <Text type="body" color="secondary" size="sm">
              {roleLabels[user.role]}
            </Text>
          </VStack>
          <Button
            isLoading={logout.isPending}
            label="Log out"
            onClick={() => {
              logout.mutate({});
            }}
            variant="ghost"
          />
        </HStack>
      }
      label="Main navigation"
      startContent={
        <>
          {can(user, "claim", "list") ? (
            <TopNavItem
              href="/claims"
              isSelected={pathname === "/claims"}
              label="My Claims"
            />
          ) : null}
          {can(user, "claim", "create") ? (
            <TopNavItem href="/claims/new" label="New Claim" />
          ) : null}
          {can(user, "review", "list") ? (
            <TopNavItem
              href="/reviews"
              isSelected={pathname === "/reviews"}
              label="Review Queue"
            />
          ) : null}
        </>
      }
    />
  );
};
