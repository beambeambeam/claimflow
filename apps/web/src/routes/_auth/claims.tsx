import { createFileRoute } from "@tanstack/react-router";

const ClaimsPage = () => (
  <main className="p-8">
    <h1 className="text-2xl font-bold">My Claims</h1>
    <p className="text-gray-600">Placeholder until F-03.</p>
  </main>
);

export const Route = createFileRoute("/_auth/claims")({
  component: ClaimsPage,
});
