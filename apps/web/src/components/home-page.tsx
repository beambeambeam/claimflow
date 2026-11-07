import { useSuspenseQuery } from "@tanstack/react-query";

import { getHealthOptions } from "#/lib/api/@tanstack/react-query.gen";

export const HomePage = () => {
  const { data: health } = useSuspenseQuery(getHealthOptions());

  return (
    <main className="p-8">
      <h1 className="text-4xl font-bold">ClaimFlow</h1>
      <p className="mt-4 text-lg">{`API version ${health.version}`}</p>
    </main>
  );
};
