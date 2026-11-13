import type { Problem } from "#/lib/api/types.gen";
import { zProblem } from "#/lib/api/zod.gen";

// The generated client throws the parsed Problem body on HTTP errors and a TypeError on network failure.
// Caught values are unknown, so the schema validates them before they are typed as Problem.
export const isProblem = (failure: unknown): failure is Problem =>
  zProblem.safeParse(failure).success;

// The first message for each field in a 400 problem's `errors`, keyed by field name (SDS §7.2).
// Other problems return no field errors.
export const problemFieldErrors = (problem: Problem) => {
  const errors = problem.status === 400 ? (problem.errors ?? []) : [];

  const firsts = errors.filter(
    (error, index) =>
      errors.findIndex((other) => other.field === error.field) === index
  );

  return Object.fromEntries(
    firsts.map(({ field, message }) => [field, { message }])
  );
};
