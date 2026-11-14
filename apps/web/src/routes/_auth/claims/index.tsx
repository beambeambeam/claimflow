import {
  createFileRoute,
  useNavigate,
  useRouteContext,
} from "@tanstack/react-router";
import { z } from "zod";

import { MyClaimsPage } from "#/features/claims/components/my-claims-page";

// A page that is not a whole number of at least 1 reads as page 1. A missing page stays missing, so `/claims` keeps its URL.
const searchSchema = z.object({
  page: z.number().int().min(1).catch(1).optional(),
});

const MyClaimsRoute = () => {
  const { user } = useRouteContext({ from: "/_auth" });
  const { page = 1 } = Route.useSearch();
  const navigate = useNavigate({ from: "/claims/" });

  return (
    <MyClaimsPage
      onPageChange={(next) => {
        void navigate({ search: { page: next } });
      }}
      page={page}
      user={user}
    />
  );
};

export const Route = createFileRoute("/_auth/claims/")({
  validateSearch: searchSchema,
  component: MyClaimsRoute,
});
