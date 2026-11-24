import { Button } from "@astryxdesign/core/Button";
import { Dialog } from "@astryxdesign/core/Dialog";
import {
  HStack,
  Layout,
  LayoutContent,
  LayoutFooter,
} from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Heading } from "@astryxdesign/core/Text";
import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useId, useState } from "react";

import { changedClaimMessage } from "#/features/claims/messages";
import {
  deleteClaimMutation,
  getClaimQueryKey,
  submitClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim, FieldError, Problem } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

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
      showToast({
        body:
          claim.status === "RETURNED"
            ? "Claim resubmitted."
            : "Claim submitted.",
      });
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

  const allowed = claim.allowedActions;
  const canEdit = allowed.includes("edit");
  const canSubmit = allowed.includes("submit");
  const canDelete = allowed.includes("delete");

  if (!canEdit && !canSubmit && !canDelete) {
    return null;
  }

  return (
    <>
      <HStack gap={2} vAlign="center">
        {canEdit ? <Link href={`/claims/${claim.id}/edit`}>Edit</Link> : null}
        {canSubmit ? (
          <Button
            isDisabled={remove.isPending}
            isLoading={submit.isPending}
            label={claim.status === "RETURNED" ? "Resubmit" : "Submit claim"}
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
      </HStack>
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
