import { Button } from "@astryxdesign/core/Button";
import { HStack } from "@astryxdesign/core/Layout";
import { Text } from "@astryxdesign/core/Text";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { DecisionDialog } from "#/components/decision-dialog";
import type {
  DecisionMutation,
  DecisionText,
} from "#/components/decision-dialog";
import {
  approveClaimMutation,
  rejectClaimMutation,
  returnClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";
import { maxReturns, returnsUsedLabel } from "#/lib/contract";

// The decisions of the assigned reviewer (F-04 §2.2, F-06 §2.1). `trigger` styles the button that opens the dialog.
// An approval of a high-value claim leaves it waiting for Finance; the toast says so from the returned status (F-08 §2.3).
const decisions: Record<
  "approve" | "reject" | "return",
  DecisionText & {
    trigger: "primary" | "secondary";
    mutation: () => DecisionMutation;
  }
> = {
  approve: {
    title: (claimNo) => `Approve claim ${claimNo}?`,
    label: "Comment (optional)",
    confirm: "Approve",
    trigger: "primary",
    variant: "primary",
    done: (decided) =>
      decided.status === "PENDING_FINANCE"
        ? "Claim approved. It now waits for a final decision by Finance."
        : "Claim approved.",
    needsReason: false,
    mutation: approveClaimMutation,
  },
  reject: {
    title: (claimNo) => `Reject claim ${claimNo}?`,
    label: "Reason",
    confirm: "Reject",
    trigger: "secondary",
    variant: "destructive",
    done: () => "Claim rejected.",
    needsReason: true,
    mutation: rejectClaimMutation,
  },
  return: {
    title: () => "Return claim",
    label: "Reason for return",
    confirm: "Return",
    trigger: "secondary",
    variant: "primary",
    done: () => "Claim returned.",
    needsReason: true,
    mutation: returnClaimMutation,
  },
};

type Decision = keyof typeof decisions;

const isDecision = (name: string): name is Decision =>
  Object.hasOwn(decisions, name);

// The Approve, Reject, and Return buttons of a claim (F-04 §2.2, F-06 §2.1). They show only for the actions the policy gave this member.
// `onDenied` runs when a decision gets a 403.
export const DecisionPanel = ({
  claim,
  onDenied,
}: {
  claim: Claim;
  onDenied: () => void;
}) => {
  const [open, setOpen] = useState<Decision | null>(null);
  const navigate = useNavigate();

  const allowed = Object.keys(decisions)
    .filter(isDecision)
    .filter((decision) => claim.allowedActions.includes(decision));

  if (allowed.length === 0) {
    return null;
  }

  const canReturn = allowed.includes("return");
  const returnsUsed = returnsUsedLabel(claim.returnCount);

  return (
    <>
      <HStack gap={2} vAlign="center">
        {allowed.map((decision) => (
          <Button
            key={decision}
            label={decisions[decision].confirm}
            onClick={() => {
              setOpen(decision);
            }}
            variant={decisions[decision].trigger}
          />
        ))}
        {canReturn ? <Text type="body">{returnsUsed}</Text> : null}
        {!canReturn && claim.returnCount >= maxReturns ? (
          <Text type="body">{`${returnsUsed} – approve or reject.`}</Text>
        ) : null}
      </HStack>
      {open === null ? null : (
        <DecisionDialog
          claim={claim}
          mutation={decisions[open].mutation()}
          onClose={() => {
            setOpen(null);
          }}
          onDecided={
            open === "return"
              ? () => {
                  // The claim left the queue, so the reviewer goes back to it (F-06 §2.1).
                  void navigate({ to: "/reviews" });
                }
              : undefined
          }
          onDenied={onDenied}
          text={decisions[open]}
        />
      )}
    </>
  );
};
