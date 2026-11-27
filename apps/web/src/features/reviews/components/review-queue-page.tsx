import { ClaimQueue } from "#/components/claim-queue";

// Review Queue (F-04 §2.1): the SUBMITTED claims assigned to the session user, oldest first, one page at a time.
export const ReviewQueuePage = ({
  page,
  onPageChange,
}: {
  page: number;
  onPageChange: (page: number) => void;
}) => (
  <ClaimQueue
    emptyTitle="No claims waiting for your review."
    onPageChange={onPageChange}
    page={page}
    scope="review"
    title="Review Queue"
  />
);
