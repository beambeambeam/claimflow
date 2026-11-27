import { Button } from "@astryxdesign/core/Button";
import { HStack } from "@astryxdesign/core/Layout";
import { useState } from "react";

import { DecisionDialog } from "#/components/decision-dialog";
import type {
  DecisionMutation,
  DecisionText,
} from "#/components/decision-dialog";
import {
  finalApproveClaimMutation,
  finalRejectClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";

// The final decisions of a Finance Officer (F-08 §2.2). There is no return from the final stage.
const decisions: Record<
  "final-approve" | "final-reject",
  DecisionText & {
    button: string;
    trigger: "primary" | "secondary";
    mutation: () => DecisionMutation;
  }
> = {
  "final-approve": {
    title: (claimNo) => `Approve claim ${claimNo} as the final decision?`,
    label: "Comment (optional)",
    confirm: "Approve",
    button: "Final approve",
    trigger: "primary",
    variant: "primary",
    done: () => "Claim approved.",
    needsReason: false,
    mutation: finalApproveClaimMutation,
  },
  "final-reject": {
    title: (claimNo) => `Reject claim ${claimNo} as the final decision?`,
    label: "Reason",
    confirm: "Reject",
    button: "Final reject",
    trigger: "secondary",
    variant: "destructive",
    done: () => "Claim rejected.",
    needsReason: true,
    mutation: finalRejectClaimMutation,
  },
};

type FinalDecision = keyof typeof decisions;

const isFinalDecision = (name: string): name is FinalDecision =>
  Object.hasOwn(decisions, name);

// The Final approve and Final reject buttons of a PENDING_FINANCE claim. They show only for the actions the policy gave this
// member. `onDenied` runs when a decision gets a 403.
export const FinalDecisionPanel = ({
  claim,
  onDenied,
}: {
  claim: Claim;
  onDenied: () => void;
}) => {
  const [open, setOpen] = useState<FinalDecision | null>(null);

  const allowed = Object.keys(decisions)
    .filter(isFinalDecision)
    .filter((decision) => claim.allowedActions.includes(decision));

  if (allowed.length === 0) {
    return null;
  }

  return (
    <>
      <HStack gap={2} vAlign="center">
        {allowed.map((decision) => (
          <Button
            key={decision}
            label={decisions[decision].button}
            onClick={() => {
              setOpen(decision);
            }}
            variant={decisions[decision].trigger}
          />
        ))}
      </HStack>
      {open === null ? null : (
        <DecisionDialog
          claim={claim}
          mutation={decisions[open].mutation()}
          onClose={() => {
            setOpen(null);
          }}
          onDenied={onDenied}
          text={decisions[open]}
        />
      )}
    </>
  );
};
