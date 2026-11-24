import { Button } from "@astryxdesign/core/Button";
import { Dialog } from "@astryxdesign/core/Dialog";
import {
  HStack,
  Layout,
  LayoutContent,
  LayoutFooter,
  VStack,
} from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Heading, Text } from "@astryxdesign/core/Text";
import { TextArea } from "@astryxdesign/core/TextArea";
import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useId, useState } from "react";

import { changedClaimMessage } from "#/features/claims/messages";
import {
  deleteClaimMutation,
  getClaimQueryKey,
  submitClaimMutation,
  withdrawClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim, FieldError, Problem } from "#/lib/api/types.gen";
import { zClaimWithdrawal } from "#/lib/api/zod.gen";
import { isProblem } from "#/lib/problem";

// The limit of the withdrawal comment, from the contract. The server checks it; the counter only shows it.
const withdrawCommentLimit =
  zClaimWithdrawal.shape.comment.unwrap().maxLength ?? undefined;

// The counter shows the limit but does not stop typing, so the text is cut at it.
const limitComment = (value: string) =>
  withdrawCommentLimit === undefined
    ? value
    : value.slice(0, withdrawCommentLimit);

const WithdrawDialog = ({
  claimNo,
  comment,
  onCancel,
  onChange,
  onWithdraw,
  withdrawing,
}: {
  claimNo: string;
  comment: string;
  onCancel: () => void;
  onChange: (comment: string) => void;
  onWithdraw: () => void;
  withdrawing: boolean;
}) => {
  const titleId = useId();

  return (
    <Dialog
      aria-labelledby={titleId}
      isOpen
      onOpenChange={(open) => {
        if (!open && !withdrawing) {
          onCancel();
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
                Withdraw {claimNo}?
              </Heading>
              <Text type="body">
                You can&apos;t edit or resubmit a withdrawn claim.
              </Text>
              <TextArea
                label="Comment (optional)"
                maxLength={withdrawCommentLimit}
                onChange={(value) => {
                  onChange(limitComment(value));
                }}
                value={comment}
              />
            </VStack>
          </LayoutContent>
        }
        footer={
          <LayoutFooter>
            <HStack gap={2} hAlign="end">
              <Button
                isDisabled={withdrawing}
                label="Cancel"
                onClick={onCancel}
                variant="ghost"
              />
              <Button
                isDisabled={withdrawing}
                isLoading={withdrawing}
                label="Withdraw claim"
                onClick={onWithdraw}
                variant="destructive"
              />
            </HStack>
          </LayoutFooter>
        }
      />
    </Dialog>
  );
};

const DeleteDraftDialog = ({
  claimNo,
  deleting,
  onCancel,
  onDelete,
}: {
  claimNo: string;
  deleting: boolean;
  onCancel: () => void;
  onDelete: () => void;
}) => {
  const titleId = useId();

  return (
    <Dialog
      aria-labelledby={titleId}
      isOpen
      onOpenChange={() => {
        if (!deleting) {
          onCancel();
        }
      }}
    >
      <Layout
        content={
          <LayoutContent>
            <Heading id={titleId} level={2}>
              Delete draft {claimNo}? This cannot be undone.
            </Heading>
          </LayoutContent>
        }
        footer={
          <LayoutFooter>
            <HStack gap={2} hAlign="end">
              <Button
                isDisabled={deleting}
                label="Cancel"
                onClick={onCancel}
                variant="ghost"
              />
              <Button
                isLoading={deleting}
                label="Delete"
                onClick={onDelete}
                variant="destructive"
              />
            </HStack>
          </LayoutFooter>
        }
      />
    </Dialog>
  );
};

export const ClaimOwnerActions = ({
  claim,
  onDenied,
  onSubmitErrors,
}: {
  claim: Claim;
  onDenied: () => void;
  onSubmitErrors: (fields: FieldError[]) => void;
}) => {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [confirmWithdraw, setConfirmWithdraw] = useState(false);
  const [withdrawComment, setWithdrawComment] = useState("");
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const showToast = useToast();
  const path = { id: claim.id };
  const detailKey = getClaimQueryKey({ path });
  const options = { path, headers: { "If-Match": `"${claim.version}"` } };

  const refreshLists = async () => {
    await queryClient.invalidateQueries({ queryKey: [{ _id: "listClaims" }] });
  };

  const handleFailure = (problem: Problem | undefined) => {
    if (problem?.status === 403) {
      onDenied();

      return;
    }

    if (problem?.status === 409) {
      showToast({
        body:
          problem.code === "CONFLICT"
            ? changedClaimMessage
            : (problem.errors?.[0]?.message ?? problem.title),
      });
      void queryClient.invalidateQueries({ queryKey: detailKey });

      return;
    }

    showToast({
      body:
        problem === undefined
          ? "Something went wrong."
          : `Something went wrong. Request ID: ${problem.requestId}`,
      type: "error",
    });
  };

  const submit = useMutation({
    ...submitClaimMutation(),
    onSuccess: async () => {
      showToast({ body: "Claim submitted." });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: detailKey }),
        refreshLists(),
      ]);
    },
    onError: (error) => {
      const problem = isProblem(error) ? error : undefined;

      if (
        problem?.status === 422 &&
        problem.errors !== undefined &&
        problem.errors.length > 0
      ) {
        onSubmitErrors(problem.errors);

        return;
      }

      handleFailure(problem);
    },
  });

  const remove = useMutation({
    ...deleteClaimMutation(),
    onSuccess: async () => {
      setConfirmDelete(false);
      void refreshLists();
      showToast({ body: "Draft deleted." });
      await navigate({ to: "/claims" });
      queryClient.removeQueries({ queryKey: detailKey });
    },
    onError: (error) => {
      setConfirmDelete(false);
      handleFailure(isProblem(error) ? error : undefined);
    },
  });

  const withdraw = useMutation({
    ...withdrawClaimMutation(),
    onSuccess: async () => {
      setConfirmWithdraw(false);
      showToast({ body: "Claim withdrawn." });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: detailKey }),
        refreshLists(),
      ]);
    },
    onError: (error) => {
      setConfirmWithdraw(false);
      handleFailure(isProblem(error) ? error : undefined);
    },
  });

  const allowed = claim.allowedActions;
  const canEdit = allowed.includes("edit");
  const canSubmit = allowed.includes("submit");
  const canDelete = allowed.includes("delete");
  const canWithdraw = allowed.includes("withdraw");

  if (!canEdit && !canSubmit && !canDelete && !canWithdraw) {
    return null;
  }

  return (
    <>
      <HStack gap={2} vAlign="center">
        {canEdit ? <Link href={`/claims/${claim.id}/edit`}>Edit</Link> : null}
        {canSubmit ? (
          <Button
            isDisabled={remove.isPending || withdraw.isPending}
            isLoading={submit.isPending}
            label="Submit claim"
            onClick={() => {
              onSubmitErrors([]);
              submit.mutate(options);
            }}
            variant="primary"
          />
        ) : null}
        {canDelete ? (
          <Button
            isDisabled={submit.isPending}
            label="Delete draft"
            onClick={() => {
              setConfirmDelete(true);
            }}
            variant="secondary"
          />
        ) : null}
        {canWithdraw ? (
          <Button
            isDisabled={submit.isPending}
            label="Withdraw"
            onClick={() => {
              setWithdrawComment("");
              setConfirmWithdraw(true);
            }}
            variant="secondary"
          />
        ) : null}
      </HStack>
      {canWithdraw && confirmWithdraw ? (
        <WithdrawDialog
          claimNo={claim.claimNo}
          comment={withdrawComment}
          onCancel={() => {
            setConfirmWithdraw(false);
          }}
          onChange={setWithdrawComment}
          onWithdraw={() => {
            withdraw.mutate({ ...options, body: { comment: withdrawComment } });
          }}
          withdrawing={withdraw.isPending}
        />
      ) : null}
      {canDelete && confirmDelete ? (
        <DeleteDraftDialog
          claimNo={claim.claimNo}
          deleting={remove.isPending}
          onCancel={() => {
            setConfirmDelete(false);
          }}
          onDelete={() => {
            remove.mutate(options);
          }}
        />
      ) : null}
    </>
  );
};
