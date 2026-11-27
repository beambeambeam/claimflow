import { Button } from "@astryxdesign/core/Button";
import type { ISODateString } from "@astryxdesign/core/Calendar";
import { Card } from "@astryxdesign/core/Card";
import { DateInput } from "@astryxdesign/core/DateInput";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { Grid } from "@astryxdesign/core/Grid";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Table } from "@astryxdesign/core/Table";
import type { TableColumn } from "@astryxdesign/core/Table";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useQuery } from "@tanstack/react-query";

import { LoadError } from "#/components/load-error";
import { StatusBadge, statusLabels } from "#/components/status-badge";
import { TextSkeleton } from "#/components/text-skeleton";
import { getDecisionSummaryOptions } from "#/lib/api/@tanstack/react-query.gen";
import type {
  ApprovedTotal,
  ExpenseCategory,
  StatusCounts,
} from "#/lib/api/types.gen";
import { formatBaht, formatMonth } from "#/lib/format";
import { categoryLabels } from "#/lib/labels";
import { isProblem } from "#/lib/problem";

export interface DateRange {
  from?: string;
  to?: string;
}

// The six statuses of a summary, in lifecycle order (F-10 §2.1).
const statuses = [
  "SUBMITTED",
  "RETURNED",
  "PENDING_FINANCE",
  "APPROVED",
  "REJECTED",
  "WITHDRAWN",
] as const satisfies readonly (keyof StatusCounts)[];

// The categories in the order of the ExpenseCategory enum, which is the order of the columns.
const categories: ExpenseCategory[] = [
  "TRAVEL",
  "ACCOMMODATION",
  "MEALS",
  "SUPPLIES",
  "TRAINING",
  "OTHER",
];

const satangOf = (total: string) => Number(total.replace(".", ""));

// The index signature lets a row be a Table row, whose type is a record.
interface MonthRow {
  [key: string]: string | number | Partial<Record<ExpenseCategory, number>>;
  month: string;
  cells: Partial<Record<ExpenseCategory, number>>;
  total: number;
}

// One row per month, in the order the API gave, with the satang of each category in it.
const monthRows = (totals: ApprovedTotal[]): MonthRow[] => {
  const rows = new Map<string, MonthRow>();

  for (const { month, category, total } of totals) {
    const row = rows.get(month) ?? { month, cells: {}, total: 0 };
    const satang = satangOf(total);
    row.cells[category] = (row.cells[category] ?? 0) + satang;
    row.total += satang;
    rows.set(month, row);
  }

  return [...rows.values()];
};

const dash = <Text type="body">–</Text>;

// The Month column, one column per category with an approved total in any month, and the month total last.
const totalsColumns = (rows: MonthRow[]): TableColumn<MonthRow>[] => [
  {
    key: "month",
    header: "Month",
    renderCell: (row) => formatMonth(row.month),
  },
  ...categories.flatMap((category): TableColumn<MonthRow>[] =>
    rows.some((row) => row.cells[category] !== undefined)
      ? [
          {
            key: category,
            header: categoryLabels[category],
            align: "end",
            renderCell: (row) => {
              const satang = row.cells[category];

              return satang === undefined ? dash : formatBaht(satang);
            },
          },
        ]
      : []
  ),
  {
    key: "total",
    header: "Total",
    align: "end",
    renderCell: (row) => formatBaht(row.total),
  },
];

const skeletonRows = Array.from({ length: 3 }, (_, index) => ({ id: index }));

const skeletonColumns: TableColumn<{ id: number }>[] = ["Month", "Total"].map(
  (header) => ({
    key: header,
    header,
    renderCell: () => <TextSkeleton type="body" />,
  })
);

const rangeMessage = "The end date must be on or after the start date.";

const isoDate = /^\d{4}-\d{2}-\d{2}$/u;

const isIsoDate = (value: string): value is ISODateString =>
  isoDate.test(value);

// The router has already dropped a search value that is not a date, so this only narrows the type for DateInput.
const asIsoDate = (value: string | undefined) =>
  value !== undefined && isIsoDate(value) ? value : undefined;

// Summary (F-10 §2.1): the current claims by status, the approved totals by month and category, and the average
// decision time, for the decisions in the date range of the URL. A member without report:read gets the access message.
export const SummaryPage = ({
  range,
  onRangeChange,
}: {
  range: DateRange;
  onRangeChange: (range: DateRange) => void;
}) => {
  const { from, to } = range;
  // The web checks the range before it sends (F-10 §2.1).
  const isBackwards = from !== undefined && to !== undefined && from > to;

  const summary = useQuery({
    ...getDecisionSummaryOptions({ query: { from, to } }),
    enabled: !isBackwards,
    retry: false,
  });

  const result = summary.data?.data;
  const forbidden = isProblem(summary.error) && summary.error.status === 403;

  const tiles = () => (
    <VStack gap={2} hAlign="stretch">
      <Grid columns={{ minWidth: 160 }} gap={4}>
        {statuses.map((status) => (
          <Card key={status}>
            <VStack gap={2} hAlign="start">
              <StatusBadge status={status} />
              {result ? (
                <Heading
                  aria-label={`${statusLabels[status]}: ${result.countsByStatus[status]}`}
                  level={2}
                >
                  {result.countsByStatus[status]}
                </Heading>
              ) : (
                <TextSkeleton type="heading-1" width="3rem" />
              )}
            </VStack>
          </Card>
        ))}
      </Grid>
      <Text type="body" color="secondary" size="sm">
        Current claims, not limited by the date range.
      </Text>
    </VStack>
  );

  const totals = () => {
    if (!result) {
      return <Table columns={skeletonColumns} data={skeletonRows} idKey="id" />;
    }

    if (result.approvedTotals.length === 0) {
      // Only REJECTED decisions in the range have an average but no approved total.
      return (
        <EmptyState
          title={
            result.averageDecisionHours === null
              ? "No decisions in this date range."
              : "No approved claims in this date range."
          }
        />
      );
    }

    const rows = monthRows(result.approvedTotals);

    return <Table columns={totalsColumns(rows)} data={rows} idKey="month" />;
  };

  const average = () => {
    if (!result) {
      return <TextSkeleton type="body" width="16rem" />;
    }

    const hours = result.averageDecisionHours;

    return (
      <Text type="body">
        {`Average decision time: ${hours === null ? "–" : `${hours.toFixed(1)} hours`}`}
      </Text>
    );
  };

  const body = () => {
    if (forbidden) {
      return (
        <Text type="body" color="secondary">
          You don&apos;t have access to this page.
        </Text>
      );
    }

    if (summary.isError) {
      return (
        <LoadError
          error={summary.error}
          onRetry={() => {
            void summary.refetch();
          }}
        />
      );
    }

    if (isBackwards) {
      return null;
    }

    return (
      <>
        {tiles()}
        <VStack gap={2} hAlign="stretch">
          <Heading level={2}>Approved totals</Heading>
          {totals()}
        </VStack>
        {average()}
      </>
    );
  };

  return (
    <VStack gap={4} hAlign="stretch">
      <Heading level={1}>Summary</Heading>
      {/* A member without report:read gets only the access message (F-10 §2.2), not the range controls. */}
      {forbidden ? null : (
        <HStack gap={4} vAlign="end">
          <DateInput
            format="system_date"
            hasClear
            label="From"
            max={asIsoDate(to)}
            onChange={(value) => {
              onRangeChange({ from: value, to });
            }}
            value={asIsoDate(from)}
          />
          <DateInput
            format="system_date"
            hasClear
            label="To"
            min={asIsoDate(from)}
            onChange={(value) => {
              onRangeChange({ from, to: value });
            }}
            status={
              isBackwards ? { type: "error", message: rangeMessage } : undefined
            }
            value={asIsoDate(to)}
          />
          {from === undefined && to === undefined ? null : (
            <Button
              label="Clear"
              onClick={() => {
                onRangeChange({});
              }}
              variant="ghost"
            />
          )}
        </HStack>
      )}
      {body()}
    </VStack>
  );
};
