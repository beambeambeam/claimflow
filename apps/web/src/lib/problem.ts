import type { Problem } from "#/lib/api/types.gen";
import { zProblem } from "#/lib/api/zod.gen";

// The generated client throws the parsed Problem body on HTTP errors and a TypeError on network failure.
// Caught values are unknown, so the schema validates them before they are typed as Problem.
export const isProblem = (failure: unknown): failure is Problem =>
  zProblem.safeParse(failure).success;
