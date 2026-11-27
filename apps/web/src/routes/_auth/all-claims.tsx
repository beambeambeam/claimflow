import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { allClaimsSearchSchema } from "#/features/claims/all-claims-search";
import { AllClaimsPage } from "#/features/claims/components/all-claims-page";

const AllClaimsRoute = () => {
  const search = Route.useSearch();
  const navigate = useNavigate({ from: "/all-claims" });

  return (
    <AllClaimsPage
      onSearchChange={(next, options) => {
        void navigate({ search: next, replace: options?.replace });
      }}
      search={search}
    />
  );
};

export const Route = createFileRoute("/_auth/all-claims")({
  validateSearch: allClaimsSearchSchema,
  component: AllClaimsRoute,
});
