import { createFileRoute } from "@tanstack/react-router";

import { ClaimDetailPage } from "#/features/claims/components/claim-detail-page";
import { DecisionPanel } from "#/features/reviews/components/decision-panel";
import type { Claim } from "#/lib/api/types.gen";

const decisionPanel = (claim: Claim) => <DecisionPanel claim={claim} />;

// The route joins the two features: Claim Detail shows the decision panel in its slot. The features do not import each other.
const ClaimDetailRoute = () => {
  const { claimId } = Route.useParams();

  return <ClaimDetailPage actions={decisionPanel} claimId={claimId} />;
};

export const Route = createFileRoute("/_auth/claims/$claimId")({
  component: ClaimDetailRoute,
});
