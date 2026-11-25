import { createFileRoute } from "@tanstack/react-router";

import { ClaimDetailPage } from "#/features/claims/components/claim-detail-page";
import { FinalDecisionPanel } from "#/features/finance/components/final-decision-panel";
import { DecisionPanel } from "#/features/reviews/components/decision-panel";
import type { Claim } from "#/lib/api/types.gen";

const decisionPanels = (claim: Claim, deny: () => void) => (
  <>
    <DecisionPanel claim={claim} onDenied={deny} />
    <FinalDecisionPanel claim={claim} onDenied={deny} />
  </>
);

// The route joins the features: Claim Detail shows the decision panels in its slot. Each panel shows only the actions
// the policy gave. The features do not import each other.
const ClaimDetailRoute = () => {
  const { claimId } = Route.useParams();

  return <ClaimDetailPage actions={decisionPanels} claimId={claimId} />;
};

export const Route = createFileRoute("/_auth/claims/$claimId")({
  component: ClaimDetailRoute,
});
