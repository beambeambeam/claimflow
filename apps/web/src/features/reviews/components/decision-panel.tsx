import { Button } from "@astryxdesign/core/Button";
import { Dialog } from "@astryxdesign/core/Dialog";
import {
  HStack,
  Layout,
  LayoutContent,
  LayoutFooter,
  VStack,
} from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";
import { TextArea } from "@astryxdesign/core/TextArea";
import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useId, useState } from "react";

import {
  approveClaimMutation,
  rejectClaimMutation,
  returnClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";
import { zClaimDecision } from "#/lib/api/zod.gen";
import {
  contractMaxLength,
  contractMinLength,
  maxReturns,
  returnsUsedLabel,
} from "#/lib/contract";
import { isProblem, problemFieldErrors } from "#/lib/problem";

// The text and button variants of each decision (F-04 §2.2, F-06 §2.1): `trigger` opens the dialog, `variant` confirms it. A reject or a return needs a reason; an approve may have a comment.
const decisions = {
  approve: {
    title: (claimNo: string) => `Approve claim ${claimNo}?`,
    label: "Comment (optional)",
    confirm: "Approve",
    trigger: "primary",
    variant: "primary",
    done: "Claim approved.",
    mutation: approveClaimMutation,
  },
  reject: {
    title: (claimNo: string) => `Reject claim ${claimNo}?`,
    label: "Reason",
    confirm: "Reject",
    trigger: "secondary",
    variant: "destructive",
    done: "Claim rejected.",
    mutation: rejectClaimMutation,
  },
  return: {
    title: () => "Return claim",
    label: "Reason for return",
    confirm: "Return",
    trigger: "secondary",
    variant: "primary",
    done: "Claim returned.",
    mutation: returnClaimMutation,
  },
} as const;

type Decision = keyof typeof decisions;

const isDecision = (name: string): name is Decision =>
  Object.hasOwn(decisions, name);

// The BR-04 metadata supplies the reject minimum and the comment counter limit.
const commentSchema = zClaimDecision.shape.comment.unwrap();
const reasonMinimum = contractMinLength(commentSchema);
const commentLimit = contractMaxLength(commentSchema);

const DecisionDialog = ({
  claim,
  decision,
  onClose,
  onDenied,
}: {
  claim: Claim;
  decision: Decision;
  onClose: () => void;
  onDenied: () => void;
}) => {
  const text = decisions[decision];
  const titleId = useId();
  const queryClient = useQueryClient();
  const showToast = useToast();
  const navigate = useNavigate();
  const [comment, setComment] = useState("");

  // The claim and the queue are read again after every answer, so they show the claim as it is now.
  const refetch = () => {
    void queryClient.invalidateQueries({ queryKey: [{ _id: "getClaim" }] });
    void queryClient.invalidateQueries({ queryKey: [{ _id: "listClaims" }] });
  };

  const send = useMutation({
    ...text.mutation(),
    onSuccess: () => {
      showToast({ body: text.done });
      onClose();
      refetch();

      if (decision === "return") {
        // The claim left the queue, so the reviewer goes back to it (F-06 §2.1).
        void navigate({ to: "/reviews" });
      }
    },
    onError: (error) => {
      const status = isProblem(error) ? error.status : null;

      if (status === 422) {
        return;
      }

      if (status === 403) {
        // The claim is no longer this member's to decide: the page shows the access message (F-04 §2.2).
        onClose();
        onDenied();

        return;
      }

      if (status === 409) {
        // Someone decided first: the reloaded claim shows the decision.
        showToast({
          body: "This claim was changed by someone else – reloaded",
        });
        onClose();
        refetch();

        return;
      }

      showToast({
        body: isProblem(error)
          ? `Something went wrong. Request ID: ${error.requestId}`
          : "Something went wrong.",
        type: "error",
      });
    },
  });

  // Match the API's Unicode whitespace trimming and code point count.
  const reasonTooShort =
    decision === "reject" &&
    (comment
      .replaceAll(
        /^\p{White_Space}+|(?<!\p{White_Space})\p{White_Space}+$/gu,
        ""
      )
      .match(/./gsu)?.length ?? 0) < reasonMinimum;

  const minimumError =
    reasonTooShort && comment !== ""
      ? `Reason must be at least ${reasonMinimum} characters.`
      : undefined;

  const serverCommentError = isProblem(send.error)
    ? problemFieldErrors(send.error).comment?.message
    : undefined;

  const commentError = minimumError ?? serverCommentError;

  // Astryx has no dialog width tokens: `width` is in pixels, and the dialog clamps it to the viewport.
  return (
    <Dialog
      aria-labelledby={titleId}
      isOpen
      onOpenChange={(open) => {
        if (!open && !send.isPending) {
          onClose();
        }
      }}
      purpose="form"
      width={480}
    >
      <Layout
        content={
          <LayoutContent>
            <VStack gap={4} hAlign="stretch">
              <Heading id={titleId} level={2}>
                {text.title(claim.claimNo)}
              </Heading>
              <TextArea
                label={text.label}
                maxLength={commentLimit}
                onChange={setComment}
                status={
                  commentError === undefined
                    ? undefined
                    : { type: "error", message: commentError }
                }
                value={comment}
              />
            </VStack>
          </LayoutContent>
        }
        footer={
          <LayoutFooter>
            <HStack gap={2} hAlign="end">
              <Button
                isDisabled={send.isPending}
                label="Cancel"
                onClick={onClose}
                variant="ghost"
              />
              <Button
                isDisabled={send.isPending || reasonTooShort}
                isLoading={send.isPending}
                label={text.confirm}
                onClick={() => {
                  send.mutate({
                    path: { id: claim.id },
                    headers: { "If-Match": `"${claim.version}"` },
                    body: { comment },
                  });
                }}
                variant={text.variant}
              />
            </HStack>
          </LayoutFooter>
        }
      />
    </Dialog>
  );
};

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
          decision={open}
          onClose={() => {
            setOpen(null);
          }}
          onDenied={onDenied}
        />
      )}
    </>
  );
};
