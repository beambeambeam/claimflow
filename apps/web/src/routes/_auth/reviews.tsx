import { createFileRoute } from "@tanstack/react-router";

const ReviewsPage = () => (
  <main className="p-8">
    <h1 className="text-2xl font-bold">Review Queue</h1>
    <p className="text-gray-600">Placeholder until F-04.</p>
  </main>
);

export const Route = createFileRoute("/_auth/reviews")({
  component: ReviewsPage,
});
