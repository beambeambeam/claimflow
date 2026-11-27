import { createFileRoute, useRouteContext } from "@tanstack/react-router";

import { NewClaimPage } from "#/features/claims/components/new-claim-page";

const NewClaimRoute = () => {
  const { user } = useRouteContext({ from: "/_auth" });

  return <NewClaimPage user={user} />;
};

export const Route = createFileRoute("/_auth/claims/new")({
  component: NewClaimRoute,
});
