import { ClaimQueue } from "#/components/claim-queue";

// Finance Queue (F-08 §2.1): every PENDING_FINANCE claim, oldest first, one page at a time.
export const FinanceQueuePage = ({
  page,
  onPageChange,
}: {
  page: number;
  onPageChange: (page: number) => void;
}) => (
  <ClaimQueue
    emptyTitle="No claims waiting for a final decision."
    onPageChange={onPageChange}
    page={page}
    scope="finance"
    title="Finance Queue"
  />
);
