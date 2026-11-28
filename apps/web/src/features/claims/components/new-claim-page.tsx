import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useRouter } from "@tanstack/react-router";
import { useEffect, useState } from "react";

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

// A link to the URL already open reloads it instead of navigating, so a page would keep its state, such as the
// confirmation of the last submit. The count changes on each such reload of pathname: use it as a key to start afresh.
export const useRemountOnSameHref = (pathname: string) => {
  const router = useRouter();
  const [visit, setVisit] = useState(0);

  useEffect(
    () =>
      router.subscribe("onBeforeNavigate", ({ hrefChanged, toLocation }) => {
        if (!hrefChanged && toLocation.pathname === pathname) {
          setVisit((count) => count + 1);
        }
      }),
    [router, pathname]
  );

  return visit;
};

// A member who has claim:create sees the New Claim form (F-02 §2.2).
export const NewClaimPage = ({ user }: { user: CurrentUser }) => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const showToast = useToast();
  const visit = useRemountOnSameHref("/claims/new");

  if (!can(user, "claim", "create")) {
    return <CannotCreateClaim />;
  }

  return (
    <NewClaimForm
      key={visit}
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
