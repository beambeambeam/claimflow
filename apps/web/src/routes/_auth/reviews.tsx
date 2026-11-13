import { createFileRoute } from "@tanstack/react-router";

import { PlaceholderPage } from "#/components/placeholder-page";

const ReviewsPage = () => (
  <PlaceholderPage note="Placeholder until F-04." title="Review Queue" />
);

export const Route = createFileRoute("/_auth/reviews")({
  component: ReviewsPage,
});
