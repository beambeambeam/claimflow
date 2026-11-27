import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { FinanceQueuePage } from "#/features/finance/components/finance-queue-page";

// A page that is not a whole number of at least 1 reads as page 1. A missing page stays missing, so `/finance` keeps its URL.
const searchSchema = z.object({
  page: z.number().int().min(1).catch(1).optional(),
});

const FinanceRoute = () => {
  const { page = 1 } = Route.useSearch();
  const navigate = useNavigate({ from: "/finance" });

  return (
    <FinanceQueuePage
      onPageChange={(next) => {
        void navigate({ search: { page: next } });
      }}
      page={page}
    />
  );
};

export const Route = createFileRoute("/_auth/finance")({
  validateSearch: searchSchema,
  component: FinanceRoute,
});
