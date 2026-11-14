import { createFileRoute } from "@tanstack/react-router";

import { ClaimDetailPage } from "#/features/claims/components/claim-detail-page";

const ClaimDetailRoute = () => {
  const { claimId } = Route.useParams();

  return <ClaimDetailPage claimId={claimId} />;
};

export const Route = createFileRoute("/_auth/claims/$claimId")({
  component: ClaimDetailRoute,
});
