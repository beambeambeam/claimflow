import { Button } from "@astryxdesign/core/Button";
import { DateInput } from "@astryxdesign/core/DateInput";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { Grid } from "@astryxdesign/core/Grid";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Pagination } from "@astryxdesign/core/Pagination";
import { Selector } from "@astryxdesign/core/Selector";
import { Table } from "@astryxdesign/core/Table";
import type { TableColumn } from "@astryxdesign/core/Table";
import { Heading } from "@astryxdesign/core/Text";
import { TextInput } from "@astryxdesign/core/TextInput";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useEffectEvent, useState } from "react";

import { DateText } from "#/components/date-text";
import { LoadError } from "#/components/load-error";
import { Money } from "#/components/money";
import { StatusBadge, statusLabels } from "#/components/status-badge";
import { TextSkeleton } from "#/components/text-skeleton";
import type {
  AllClaimsFilters,
  AllClaimsSearch,
} from "#/features/claims/all-claims-search";
import { listClaimsOptions } from "#/lib/api/@tanstack/react-query.gen";
import type { ClaimSummary } from "#/lib/api/types.gen";
import { zClaimStatus, zExpenseCategory } from "#/lib/api/zod.gen";
import { isBackwardsRange, rangeMessage } from "#/lib/date-range";
import { categoryLabels } from "#/lib/labels";
import { isProblem, problemFieldErrors } from "#/lib/problem";

// The page size of the list (F-03 §1.4 default).
const pageSize = 20;

// How long the search waits after the last key press (F-09 §2.1).
const searchDelay = 300;

// The longest search text the API accepts (F-09 §1.3).
const searchMaxLength = 120;

// The value of the "All" option. It is not a status or a category, so it parses to no filter.
const allValue = "ALL";

const optionsOf = (labels: Record<string, string>) => [
  { value: allValue, label: "All" },
  ...Object.entries(labels).map(([value, label]) => ({ value, label })),
];

const statusOptions = optionsOf(statusLabels);

const categoryOptions = optionsOf(categoryLabels);

const searchLengthMessage = "Search text must be at most 120 characters.";

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
    key: "status",
    header: "Status",
    renderCell: (claim) => <StatusBadge status={claim.status} />,
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

// The same columns with a placeholder in each cell, shown while the list loads.
const skeletonColumns: TableColumn<{ id: number }>[] = columns.map(
  ({ key, header, align }) => ({
    key,
    header,
    align,
    renderCell: () => <TextSkeleton type="body" />,
  })
);

const errorStatus = (message: string | undefined) =>
  message === undefined ? undefined : { type: "error" as const, message };

// The search-param form of the typed text: trimmed, and no filter when blank.
const searchOf = (text: string) => {
  const trimmed = text.trim();

  return trimmed === "" ? undefined : trimmed;
};

// The API counts characters as code points, which is what spreading a string yields.
const isTooLong = (search: string) =>
  // oxlint-disable-next-line typescript/no-misused-spread
  [...search].length > searchMaxLength;

type FieldErrors = Partial<Record<string, { message: string }>>;

// The search box. It keeps the typed text and sends it once typing pauses for the search delay.
const SearchInput = ({
  q,
  message,
  onSearch,
}: {
  q: string | undefined;
  message: string | undefined;
  onSearch: (q: string | undefined) => void;
}) => {
  const [text, setText] = useState(q ?? "");
  const [shownQ, setShownQ] = useState(q);
  const typed = searchOf(text);
  const tooLong = typed !== undefined && isTooLong(typed);
  const isChanged = typed !== q && !tooLong;

  // Follow `q` when it changes outside the input, as after Clear filters or a history step.
  if (shownQ !== q) {
    setShownQ(q);

    if (typed !== q) {
      setText(q ?? "");
    }
  }

  const send = useEffectEvent(onSearch);

  // Each key press restarts the wait, so only the text after the last key is sent.
  useEffect(() => {
    const timer = setTimeout(() => {
      if (isChanged) {
        send(typed);
      }
    }, searchDelay);

    return () => {
      clearTimeout(timer);
    };
  }, [isChanged, typed]);

  return (
    <TextInput
      label="Search title"
      onChange={setText}
      status={errorStatus(tooLong ? searchLengthMessage : message)}
      type="search"
      value={text}
    />
  );
};

// The filter bar (F-09 §2.1). Each control replaces its own filter.
const FilterBar = ({
  filters,
  fieldErrors,
  isBadRange,
  onChange,
}: {
  filters: AllClaimsFilters;
  fieldErrors: FieldErrors;
  isBadRange: boolean;
  onChange: (change: AllClaimsFilters) => void;
}) => (
  <Grid align="start" columns={{ minWidth: 180, max: 5 }} gap={3}>
    <Selector
      label="Status"
      onChange={(value) => {
        onChange({ status: zClaimStatus.safeParse(value).data });
      }}
      options={statusOptions}
      status={errorStatus(fieldErrors.status?.message)}
      value={filters.status ?? allValue}
    />
    <Selector
      label="Category"
      onChange={(value) => {
        onChange({ category: zExpenseCategory.safeParse(value).data });
      }}
      options={categoryOptions}
      status={errorStatus(fieldErrors.category?.message)}
      value={filters.category ?? allValue}
    />
    <DateInput
      label="From"
      onChange={(from) => {
        onChange({ from });
      }}
      status={errorStatus(fieldErrors.from?.message)}
      value={filters.from}
    />
    <DateInput
      label="To"
      onChange={(to) => {
        onChange({ to });
      }}
      status={errorStatus(isBadRange ? rangeMessage : fieldErrors.to?.message)}
      value={filters.to}
    />
    <SearchInput
      message={fieldErrors.q?.message}
      onSearch={(q) => {
        onChange({ q });
      }}
      q={filters.q}
    />
  </Grid>
);

// All Claims (F-09 §2.1): every claim the session user can see, newest first, narrowed by the filters in the URL.
// A filter change goes back to page 1.
export const AllClaimsPage = ({
  search,
  onSearchChange,
}: {
  search: AllClaimsSearch;
  onSearchChange: (
    search: AllClaimsSearch,
    options?: { replace: boolean }
  ) => void;
}) => {
  const { page = 1, ...filters } = search;

  const isFiltered = Object.values(filters).some(
    (value) => value !== undefined
  );

  // The API would reject the range, so it is never sent.
  const isBadRange = isBackwardsRange(filters.from, filters.to);

  const claims = useQuery({
    ...listClaimsOptions({
      query: { scope: "all", ...filters, page, pageSize },
    }),
    enabled: !isBadRange,
    retry: false,
  });

  const clearFilters = () => {
    onSearchChange({});
  };

  const total = claims.data?.total ?? 0;
  const lastPage = Math.ceil(total / pageSize);
  const pastLastPage = total > 0 && page > lastPage;

  const showLastPage = useEffectEvent(() => {
    onSearchChange({ ...filters, page: lastPage }, { replace: true });
  });

  useEffect(() => {
    if (pastLastPage) {
      showLastPage();
    }
  }, [pastLastPage]);

  const fieldErrors: FieldErrors = isProblem(claims.error)
    ? problemFieldErrors(claims.error)
    : {};

  const body = () => {
    // The controls show what is wrong, so the list shows nothing.
    if (isBadRange || Object.keys(fieldErrors).length > 0) {
      return null;
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
      return isFiltered ? (
        <EmptyState
          actions={<Button label="Clear filters" onClick={clearFilters} />}
          title="No claims match these filters."
        />
      ) : (
        <EmptyState title="No claims yet." />
      );
    }

    return (
      <>
        <Table columns={columns} data={claims.data.data} idKey="id" />
        <HStack hAlign="center" gap={4} vAlign="center">
          <Pagination
            onChange={(next) => {
              onSearchChange({ ...filters, page: next });
            }}
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
        <Heading level={1}>All Claims</Heading>
        {isFiltered ? (
          <Button
            label="Clear filters"
            onClick={clearFilters}
            variant="ghost"
          />
        ) : null}
      </HStack>
      <FilterBar
        fieldErrors={fieldErrors}
        filters={filters}
        isBadRange={isBadRange}
        onChange={(change) => {
          onSearchChange({ ...filters, ...change, page: 1 });
        }}
      />
      {body()}
    </VStack>
  );
};
