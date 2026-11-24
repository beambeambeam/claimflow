import { createFileRoute } from "@tanstack/react-router";

import { EditClaimPage } from "#/features/claims/components/edit-claim-page";

const EditClaimRoute = () => {
  const { claimId } = Route.useParams();

  return <EditClaimPage claimId={claimId} />;
};

export const Route = createFileRoute("/_auth/claims/$claimId_/edit")({
  component: EditClaimRoute,
});
