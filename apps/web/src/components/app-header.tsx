import { Button } from "@astryxdesign/core/Button";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Text } from "@astryxdesign/core/Text";
import { TopNav, TopNavItem } from "@astryxdesign/core/TopNav";
import { useRouterState } from "@tanstack/react-router";

import type { CurrentUser } from "#/lib/api/types.gen";

type Role = CurrentUser["role"];

// Role labels from F-01 §2.2.
const roleLabels: Record<Role, string> = {
  requester: "Requester",
  reviewer: "Reviewer",
  finance: "Finance Officer",
};

// Links are gated by permission, not role name (F-01 §1.6).
const can = (user: CurrentUser, resource: string, action: string) =>
  user.permissions[resource]?.includes(action);

export const AppHeader = ({ user }: { user: CurrentUser }) => {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
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
          <Button label="Log out" variant="ghost" />
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
