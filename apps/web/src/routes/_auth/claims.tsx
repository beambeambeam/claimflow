import { createFileRoute } from "@tanstack/react-router";

import { PlaceholderPage } from "#/components/placeholder-page";

const ClaimsPage = () => (
  <PlaceholderPage note="Placeholder until F-03." title="My Claims" />
);

export const Route = createFileRoute("/_auth/claims")({
  component: ClaimsPage,
});
