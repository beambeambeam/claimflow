import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { ReviewQueuePage } from "#/features/reviews/components/review-queue-page";

// A page that is not a whole number of at least 1 reads as page 1. A missing page stays missing, so `/reviews` keeps its URL.
const searchSchema = z.object({
  page: z.number().int().min(1).catch(1).optional(),
});

const ReviewsRoute = () => {
  const { page = 1 } = Route.useSearch();
  const navigate = useNavigate({ from: "/reviews" });

  return (
    <ReviewQueuePage
      onPageChange={(next) => {
        void navigate({ search: { page: next } });
      }}
      page={page}
    />
  );
};

export const Route = createFileRoute("/_auth/reviews")({
  validateSearch: searchSchema,
  component: ReviewsRoute,
});
