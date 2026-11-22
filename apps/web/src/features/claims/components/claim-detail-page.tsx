import { HStack, VStack } from "@astryxdesign/core/Layout";
import { List, ListItem } from "@astryxdesign/core/List";
import {
  MetadataList,
  MetadataListItem,
} from "@astryxdesign/core/MetadataList";
import { Table } from "@astryxdesign/core/Table";
import type { TableColumn } from "@astryxdesign/core/Table";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { DateText } from "#/components/date-text";
import { LoadError } from "#/components/load-error";
import { Money } from "#/components/money";
import { StatusBadge, statusLabels } from "#/components/status-badge";
import { TextSkeleton } from "#/components/text-skeleton";
import { actionLabels, categoryLabels } from "#/features/claims/labels";
import { getClaimOptions } from "#/lib/api/@tanstack/react-query.gen";
import type { Claim, HistoryEntry, LineItem } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

const itemColumns: TableColumn<LineItem>[] = [
  { key: "lineNo", header: "#" },
  {
    key: "category",
    header: "Category",
    renderCell: (item) => categoryLabels[item.category],
  },
  { key: "description", header: "Description" },
  {
    key: "expenseDate",
    header: "Expense date",
    renderCell: (item) => <DateText value={item.expenseDate} />,
  },
  {
    key: "amount",
    header: "Amount",
    align: "end",
    renderCell: (item) => <Money value={item.amount} />,
  },
  { key: "receiptRef", header: "Receipt ref" },
];

const skeletonRows = Array.from({ length: 3 }, (_, index) => ({ id: index }));

const skeletonColumns: TableColumn<{ id: number }>[] = itemColumns.map(
  ({ key, header, align }) => ({
    key,
    header,
    align,
    renderCell: () => <TextSkeleton type="body" />,
  })
);

const Section = ({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) => (
  <VStack aria-label={title} as="section" gap={2} hAlign="stretch">
    <Heading level={2}>{title}</Heading>
    {children}
  </VStack>
);

// "Submitted → Approved". The first entry has no earlier status, so it starts with a dash.
const transitionText = ({ fromStatus, toStatus }: HistoryEntry) =>
  `${fromStatus === null ? "–" : statusLabels[fromStatus]} → ${statusLabels[toStatus]}`;

const HistoryTimeline = ({ history }: { history: HistoryEntry[] }) => (
  <List hasDividers>
    {history.map((entry) => (
      <ListItem
        description={
          <VStack gap={1}>
            <Text type="body" color="secondary">
              {transitionText(entry)}
            </Text>
            <HStack gap={2}>
              <Text type="body">{entry.actor.name}</Text>
              <DateText value={entry.createdAt} withTime />
            </HStack>
            {entry.comment === null ? null : (
              <Text type="body">{entry.comment}</Text>
            )}
          </VStack>
        }
        itemKey={entry.id}
        key={entry.id}
        label={actionLabels[entry.action]}
      />
    ))}
  </List>
);

// The slot under the claim header. `deny` swaps the page for the access message, as when a decision gets a 403.
type ClaimActions = (claim: Claim, deny: () => void) => React.ReactNode;

const ClaimView = ({
  claim,
  actions,
  deny,
}: {
  claim: Claim;
  actions?: ClaimActions;
  deny: () => void;
}) => (
  <VStack gap={6} hAlign="stretch">
    <VStack aria-label="Claim" as="section" gap={2}>
      <HStack gap={3} vAlign="center">
        <Heading level={1}>{claim.claimNo}</Heading>
        <StatusBadge status={claim.status} />
      </HStack>
      <Text type="large">{claim.title}</Text>
      <Money value={claim.total} />
      {actions?.(claim, deny)}
    </VStack>

    <Section title="Summary">
      <MetadataList>
        <MetadataListItem label="Purpose">{claim.purpose}</MetadataListItem>
        <MetadataListItem label="Requester">
          {claim.requester.name}
        </MetadataListItem>
        <MetadataListItem label="Assigned reviewer">
          {claim.assignedReviewer?.name ?? "–"}
        </MetadataListItem>
        <MetadataListItem label="Submitted">
          <DateText value={claim.submittedAt} />
        </MetadataListItem>
      </MetadataList>
    </Section>

    <Section title="Line items">
      <Table columns={itemColumns} data={claim.items} idKey="lineNo" />
    </Section>

    <Section title="History">
      <HistoryTimeline history={claim.history} />
    </Section>
  </VStack>
);

const ClaimSkeleton = () => (
  <VStack gap={6} hAlign="stretch">
    <VStack gap={2}>
      <TextSkeleton type="heading-1" width="15ch" />
      <TextSkeleton type="large" width="60%" />
    </VStack>
    <Table columns={skeletonColumns} data={skeletonRows} idKey="id" />
  </VStack>
);

const AccessDenied = () => (
  <Text type="body" color="secondary">
    You don&apos;t have access to this claim.
  </Text>
);

// Claim Detail (F-03 §2.2). A claim the member may not see, or that does not exist, shows its message and no claim data.
// `actions` is a slot under the claim header, which the route fills with what a feature adds, such as the decision panel.
// A 403 from an action shows the same access message as a 403 from the read (F-04 §2.2).
export const ClaimDetailPage = ({
  claimId,
  actions,
}: {
  claimId: string;
  actions?: ClaimActions;
}) => {
  const [denied, setDenied] = useState(false);

  const claim = useQuery({
    ...getClaimOptions({ path: { id: claimId } }),
    retry: false,
  });

  if (denied) {
    return <AccessDenied />;
  }

  if (claim.isPending) {
    return <ClaimSkeleton />;
  }

  if (claim.isError) {
    const status = isProblem(claim.error) ? claim.error.status : null;

    if (status === 403) {
      return <AccessDenied />;
    }

    if (status === 404) {
      return (
        <Text type="body" color="secondary">
          Claim not found.
        </Text>
      );
    }

    return (
      <LoadError
        error={claim.error}
        onRetry={() => {
          void claim.refetch();
        }}
      />
    );
  }

  return (
    <ClaimView
      actions={actions}
      claim={claim.data.data}
      deny={() => {
        setDenied(true);
      }}
    />
  );
};
