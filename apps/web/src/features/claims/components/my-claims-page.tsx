import { Button } from "@astryxdesign/core/Button";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Pagination } from "@astryxdesign/core/Pagination";
import { Table } from "@astryxdesign/core/Table";
import type { TableColumn } from "@astryxdesign/core/Table";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { DateText } from "#/components/date-text";
import { LoadError } from "#/components/load-error";
import { Money } from "#/components/money";
import { StatusBadge } from "#/components/status-badge";
import { TextSkeleton } from "#/components/text-skeleton";
import { can } from "#/features/auth/permissions";
import { listClaimsOptions } from "#/lib/api/@tanstack/react-query.gen";
import type { ClaimSummary, CurrentUser } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

// The page size of the list (F-03 §1.4 default).
const pageSize = 20;

const skeletonRows = Array.from({ length: 5 }, (_, index) => ({ id: index }));

const claimLink = (claim: ClaimSummary, label: string) => (
  <Link href={`/claims/${claim.id}`}>{label}</Link>
);

const columns: TableColumn<ClaimSummary>[] = [
  {
    key: "claimNo",
    header: "Claim no.",
    renderCell: (claim) => claimLink(claim, claim.claimNo),
  },
  {
    key: "title",
    header: "Title",
    renderCell: (claim) => claimLink(claim, claim.title),
  },
  {
    key: "total",
    header: "Total",
    align: "end",
    renderCell: (claim) => <Money value={claim.total} />,
  },
  {
    key: "status",
    header: "Status",
    renderCell: (claim) => <StatusBadge status={claim.status} />,
  },
  {
    key: "submittedAt",
    header: "Submitted",
    renderCell: (claim) => <DateText value={claim.submittedAt} />,
  },
];

// The same columns with a placeholder in each cell, shown while the list loads.
const skeletonColumns: TableColumn<{ id: number }>[] = columns.map(
  ({ key, header, align }) => ({
    key,
    header,
    align,
    renderCell: () => <TextSkeleton type="body" />,
  })
);

// My Claims (F-03 §2.1): the session user's claims, newest first, one page at a time. A page past the last one is
// replaced by the last page, so the table never shows an empty page above the pager.
export const MyClaimsPage = ({
  user,
  page,
  onPageChange,
}: {
  user: CurrentUser;
  page: number;
  onPageChange: (page: number, options?: { replace: boolean }) => void;
}) => {
  const claims = useQuery({
    ...listClaimsOptions({ query: { scope: "mine", page, pageSize } }),
    retry: false,
  });

  const total = claims.data?.total ?? 0;
  const lastPage = Math.ceil(total / pageSize);
  const pastLastPage = total > 0 && page > lastPage;

  useEffect(() => {
    if (pastLastPage) {
      onPageChange(lastPage, { replace: true });
    }
  }, [lastPage, onPageChange, pastLastPage]);

  const forbidden = isProblem(claims.error) && claims.error.status === 403;

  const body = () => {
    if (forbidden) {
      return (
        <Text type="body" color="secondary">
          You don&apos;t have access to this page.
        </Text>
      );
    }

    if (claims.isError) {
      return (
        <LoadError
          error={claims.error}
          onRetry={() => {
            void claims.refetch();
          }}
        />
      );
    }

    if (claims.isPending || pastLastPage) {
      return <Table columns={skeletonColumns} data={skeletonRows} idKey="id" />;
    }

    if (claims.data.total === 0) {
      return (
        <EmptyState
          actions={<Button href="/claims/new" label="Create claim" />}
          title="No claims yet"
        />
      );
    }

    return (
      <>
        <Table columns={columns} data={claims.data.data} idKey="id" />
        <HStack hAlign="center" gap={4} vAlign="center">
          <Pagination
            onChange={onPageChange}
            page={page}
            pageSize={pageSize}
            totalItems={claims.data.total}
            variant="compact"
          />
        </HStack>
      </>
    );
  };

  return (
    <VStack gap={4} hAlign="stretch">
      <HStack gap={4} hAlign="between" vAlign="center">
        <Heading level={1}>My Claims</Heading>
        {can(user, "claim", "create") ? (
          <Button href="/claims/new" label="New claim" variant="primary" />
        ) : null}
      </HStack>
      {body()}
    </VStack>
  );
};
