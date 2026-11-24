import { VStack } from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Text } from "@astryxdesign/core/Text";
import { useToast } from "@astryxdesign/core/Toast";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { TextSkeleton } from "#/components/text-skeleton";
import { changedClaimMessage } from "#/features/claims/messages";
import {
  getClaimOptions,
  getClaimQueryKey,
  updateClaimMutation,
} from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

import { ClaimForm } from "./claim-form";
import { ClaimLoadFailure } from "./claim-load-failure";

const EditClaimForm = ({
  claim,
  reload,
}: {
  claim: Claim;
  reload: () => Promise<Claim | undefined>;
}) => {
  const [loaded, setLoaded] = useState(claim);
  const navigate = useNavigate();
  const showToast = useToast();
  const queryClient = useQueryClient();
  const edit = useMutation(updateClaimMutation());
  const path = { id: claim.id };

  const goToDetail = async () => {
    await navigate({ to: "/claims/$claimId", params: { claimId: claim.id } });
  };

  if (edit.error?.status === 403 || edit.error?.status === 404) {
    return (
      <ClaimLoadFailure
        error={edit.error}
        onRetry={() => {
          edit.reset();
        }}
      />
    );
  }

  return (
    <ClaimForm
      cancelHref={`/claims/${claim.id}`}
      error={edit.error?.status === 409 ? null : edit.error}
      initialValues={{
        title: loaded.title,
        purpose: loaded.purpose,
        items: loaded.items,
      }}
      key={loaded.version}
      onSave={async (body) => {
        try {
          const result = await edit.mutateAsync({
            path,
            body,
            headers: { "If-Match": `"${loaded.version}"` },
          });

          queryClient.setQueryData(getClaimQueryKey({ path }), result);
          void queryClient.invalidateQueries({
            queryKey: [{ _id: "listClaims" }],
          });
          showToast({ body: "Changes saved." });
          await goToDetail();
        } catch (error) {
          if (!isProblem(error) || error.status !== 409) {
            throw error;
          }

          if (error.code === "CONFLICT") {
            showToast({ body: changedClaimMessage });
            const latest = await reload();

            if (latest !== undefined) {
              setLoaded(latest);
            }

            edit.reset();

            return;
          }

          showToast({ body: error.errors?.[0]?.message ?? error.title });
          void queryClient.invalidateQueries({
            queryKey: getClaimQueryKey({ path }),
          });
          await goToDetail();
        }
      }}
    />
  );
};

export const EditClaimPage = ({ claimId }: { claimId: string }) => {
  const query = useQuery({
    ...getClaimOptions({ path: { id: claimId } }),
    retry: false,
  });

  const reload = async () => {
    const latest = await query.refetch();

    return latest.data?.data;
  };

  if (query.isPending) {
    return <TextSkeleton type="heading-1" width="15ch" />;
  }

  if (query.isError) {
    return (
      <ClaimLoadFailure
        error={query.error}
        onRetry={() => {
          void reload();
        }}
      />
    );
  }

  if (!query.data.data.allowedActions.includes("edit")) {
    return (
      <VStack gap={2}>
        <Text type="body">This claim can&apos;t be edited.</Text>
        <Link href={`/claims/${claimId}`}>Back to claim</Link>
      </VStack>
    );
  }

  return (
    <EditClaimForm claim={query.data.data} key={claimId} reload={reload} />
  );
};
