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
import type { UseMutationOptions } from "@tanstack/react-query";
import { useId, useState } from "react";

import type { Options } from "#/lib/api/sdk.gen";
import type {
  ApproveClaimData,
  Claim,
  ClaimEnvelope,
  Problem,
} from "#/lib/api/types.gen";
import { zClaimDecision } from "#/lib/api/zod.gen";
import { contractMaxLength, contractMinLength } from "#/lib/contract";
import { isProblem, problemFieldErrors } from "#/lib/problem";

// The text of one decision (F-04 §2.2, F-06 §2.1, F-08 §2.2). `variant` styles the confirm button, `done` is the
// success toast for the decided claim, and `needsReason` marks the decisions that the API refuses without a reason.
export interface DecisionText {
  title: (claimNo: string) => string;
  label: string;
  confirm: string;
  variant: "primary" | "destructive";
  done: (decided: Claim) => string;
  needsReason: boolean;
}

// Every decision operation takes the same request: the claim id, If-Match, and a ClaimDecision body.
type DecisionRequest = Options<ApproveClaimData>;

export type DecisionMutation = UseMutationOptions<
  ClaimEnvelope,
  Problem,
  DecisionRequest
>;

// The BR-04 metadata supplies the reason minimum and the comment counter limit.
const commentSchema = zClaimDecision.shape.comment.unwrap();
const reasonMinimum = contractMinLength(commentSchema);
const commentLimit = contractMaxLength(commentSchema);

// The confirmation dialog of a decision. It sends the comment with the claim's version, then reads the claim and the
// lists again. A 422 stays in the dialog under the field, a 403 runs `onDenied`, and a 409 reloads the claim.
export const DecisionDialog = ({
  claim,
  text,
  mutation,
  onDecided,
  onClose,
  onDenied,
}: {
  claim: Claim;
  text: DecisionText;
  mutation: DecisionMutation;
  onDecided?: () => void;
  onClose: () => void;
  onDenied: () => void;
}) => {
  const titleId = useId();
  const queryClient = useQueryClient();
  const showToast = useToast();
  const [comment, setComment] = useState("");

  // The claim and the queues are read again after every answer, so they show the claim as it is now.
  const refetch = () => {
    void queryClient.invalidateQueries({ queryKey: [{ _id: "getClaim" }] });
    void queryClient.invalidateQueries({ queryKey: [{ _id: "listClaims" }] });
  };

  const send = useMutation({
    ...mutation,
    onSuccess: (decided) => {
      showToast({ body: text.done(decided.data) });
      onClose();
      refetch();
      onDecided?.();
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
    text.needsReason &&
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
