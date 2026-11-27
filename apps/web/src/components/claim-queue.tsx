import { EmptyState } from "@astryxdesign/core/EmptyState";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Pagination } from "@astryxdesign/core/Pagination";
import { Table } from "@astryxdesign/core/Table";
import type { TableColumn } from "@astryxdesign/core/Table";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useQuery } from "@tanstack/react-query";

import { DateText } from "#/components/date-text";
import { LoadError } from "#/components/load-error";
import { Money } from "#/components/money";
import { TextSkeleton } from "#/components/text-skeleton";
import { listClaimsOptions } from "#/lib/api/@tanstack/react-query.gen";
import type { ClaimSummary, ListClaimsData } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

// The page size of the queue (F-03 §1.4 default).
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
    key: "requester",
    header: "Requester",
    renderCell: (claim) => claim.requester.name,
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
    key: "submittedAt",
    header: "Submitted",
    renderCell: (claim) => <DateText value={claim.submittedAt} />,
  },
];

// The same columns with a placeholder in each cell, shown while the queue loads.
const skeletonColumns: TableColumn<{ id: number }>[] = columns.map(
  ({ key, header, align }) => ({
    key,
    header,
    align,
    renderCell: () => <TextSkeleton type="body" />,
  })
);

// A queue of claims waiting for a decision, oldest first, one page at a time: the Review Queue (F-04 §2.1) and the
// Finance Queue (F-08 §2.1). A row opens Claim Detail. A member without the scope's route permission gets the access
// message.
export const ClaimQueue = ({
  title,
  scope,
  emptyTitle,
  page,
  onPageChange,
}: {
  title: string;
  scope: ListClaimsData["query"]["scope"];
  emptyTitle: string;
  page: number;
  onPageChange: (page: number) => void;
}) => {
  const claims = useQuery({
    ...listClaimsOptions({ query: { scope, page, pageSize } }),
    retry: false,
  });

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

    if (claims.isPending) {
      return <Table columns={skeletonColumns} data={skeletonRows} idKey="id" />;
    }

    if (claims.data.total === 0) {
      return <EmptyState title={emptyTitle} />;
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
      <Heading level={1}>{title}</Heading>
      {body()}
    </VStack>
  );
};
