import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { SummaryPage } from "#/features/reports/components/summary-page";

// A date that is not a calendar date YYYY-MM-DD is dropped, so a hand-edited URL never sends a bad range.
const searchSchema = z.object({
  from: z.iso.date().optional().catch(undefined),
  to: z.iso.date().optional().catch(undefined),
});

const ReportsRoute = () => {
  const range = Route.useSearch();
  const navigate = useNavigate({ from: "/reports" });

  return (
    <SummaryPage
      onRangeChange={(next) => {
        void navigate({ replace: true, search: next });
      }}
      range={range}
    />
  );
};

export const Route = createFileRoute("/_auth/reports")({
  validateSearch: searchSchema,
  component: ReportsRoute,
});
