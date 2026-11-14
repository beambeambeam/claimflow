import { Button } from "@astryxdesign/core/Button";
import { Dialog } from "@astryxdesign/core/Dialog";
import {
  HStack,
  Layout,
  LayoutContent,
  LayoutFooter,
  VStack,
} from "@astryxdesign/core/Layout";
import { Heading } from "@astryxdesign/core/Text";
import { TextArea } from "@astryxdesign/core/TextArea";
import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useId, useState } from "react";

import {
  approveClaimMutation,
  rejectClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";
import { isProblem, problemFieldErrors } from "#/lib/problem";

type Decision = "approve" | "reject";

// The text of each decision (F-04 §2.2). A reject needs a reason; an approve may have a comment.
const decisions = {
  approve: {
    title: (claimNo: string) => `Approve claim ${claimNo}?`,
    label: "Comment (optional)",
    confirm: "Approve",
    variant: "primary",
    done: "Claim approved.",
    mutation: approveClaimMutation,
  },
  reject: {
    title: (claimNo: string) => `Reject claim ${claimNo}?`,
    label: "Reason",
    confirm: "Reject",
    variant: "destructive",
    done: "Claim rejected.",
    mutation: rejectClaimMutation,
  },
} as const;

// The limit of BR-04. The server checks it; the counter only shows it.
const commentLimit = 500;

const DecisionDialog = ({
  claim,
  decision,
  onClose,
}: {
  claim: Claim;
  decision: Decision;
  onClose: () => void;
}) => {
  const text = decisions[decision];
  const titleId = useId();
  const queryClient = useQueryClient();
  const showToast = useToast();
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
    },
    onError: (error) => {
      const status = isProblem(error) ? error.status : null;

      if (status === 422) {
        return;
      }

      if (status === 409 || status === 403) {
        // Someone decided first, or the claim is no longer this member's: the reloaded claim tells which.
        if (status === 409) {
          showToast({
            body: "This claim was changed by someone else – reloaded",
          });
        }

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

  const reasonError = isProblem(send.error)
    ? problemFieldErrors(send.error).comment?.message
    : undefined;

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
                  reasonError === undefined
                    ? undefined
                    : { type: "error", message: reasonError }
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
                isDisabled={send.isPending}
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

// The Approve and Reject buttons of a claim (F-04 §2.2). They show only for the actions the policy gave this member.
export const DecisionPanel = ({ claim }: { claim: Claim }) => {
  const [open, setOpen] = useState<Decision | null>(null);

  const allowed = (["approve", "reject"] as const).filter((decision) =>
    claim.allowedActions.includes(decision)
  );

  if (allowed.length === 0) {
    return null;
  }

  return (
    <>
      <HStack gap={2}>
        {allowed.map((decision) => (
          <Button
            key={decision}
            label={decisions[decision].confirm}
            onClick={() => {
              setOpen(decision);
            }}
            variant={decision === "approve" ? "primary" : "secondary"}
          />
        ))}
      </HStack>
      {open === null ? null : (
        <DecisionDialog
          claim={claim}
          decision={open}
          onClose={() => {
            setOpen(null);
          }}
        />
      )}
    </>
  );
};
