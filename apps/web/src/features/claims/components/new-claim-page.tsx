import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { can } from "#/features/auth/permissions";
import {
  createClaimMutation,
  getClaimQueryKey,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim, ClaimEnvelope, CurrentUser } from "#/lib/api/types.gen";

import { CannotCreateClaim } from "./cannot-create-claim";
import { ClaimConfirmation } from "./claim-confirmation";
import { ClaimForm } from "./claim-form";

export const NewClaimForm = ({
  onDraftSaved,
}: {
  onDraftSaved: (draft: ClaimEnvelope) => void;
}) => {
  const [submitted, setSubmitted] = useState<Claim | null>(null);
  const create = useMutation(createClaimMutation());

  if (submitted !== null) {
    return (
      <ClaimConfirmation
        claim={submitted}
        onNewClaim={() => {
          create.reset();
          setSubmitted(null);
        }}
      />
    );
  }

  return (
    <ClaimForm
      error={create.error}
      onSave={async (body, intent) => {
        if (intent === "draft") {
          const draft = await create.mutateAsync({
            body,
            query: { draft: true },
          });

          onDraftSaved(draft);

          return;
        }

        const envelope = await create.mutateAsync({ body });
        setSubmitted(envelope.data);
      }}
    />
  );
};

// A member who has claim:create sees the New Claim form (F-02 §2.2).
export const NewClaimPage = ({ user }: { user: CurrentUser }) => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const showToast = useToast();

  if (!can(user, "claim", "create")) {
    return <CannotCreateClaim />;
  }

  return (
    <NewClaimForm
      onDraftSaved={(draft) => {
        const claimId = draft.data.id;
        queryClient.setQueryData(
          getClaimQueryKey({ path: { id: claimId } }),
          draft
        );
        void queryClient.invalidateQueries({
          queryKey: [{ _id: "listClaims" }],
        });
        showToast({ body: "Draft saved." });
        void navigate({ to: "/claims/$claimId", params: { claimId } });
      }}
    />
  );
};
