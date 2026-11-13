import type { Problem } from "#/lib/api/types.gen";
import { zProblem } from "#/lib/api/zod.gen";

// The generated client throws the parsed Problem body on HTTP errors and a TypeError on network failure.
export type RequestFailure = Problem | TypeError;

export const readProblem = (failure: RequestFailure | null): Problem | null => {
  const parsed = zProblem.safeParse(failure);

  return parsed.success ? parsed.data : null;
};
