import type { ISODateString } from "@astryxdesign/core/Calendar";
import { z } from "zod";

import { zClaimStatus, zExpenseCategory } from "#/lib/api/zod.gen";

const isoDateSchema = z.iso.date();

// A `YYYY-MM-DD` calendar date, typed as the date DateInput takes.
const isoDate = z.custom<ISODateString>(
  (value) => isoDateSchema.safeParse(value).success
);

// The All Claims search params (F-09 §2.1). A value the API would reject is dropped, so a hand-edited URL opens the
// page without that filter. `from` later than `to` stays, and the page shows the range error instead of sending it.
export const allClaimsSearchSchema = z.object({
  status: zClaimStatus.optional().catch(undefined),
  category: zExpenseCategory.optional().catch(undefined),
  from: isoDate.optional().catch(undefined),
  to: isoDate.optional().catch(undefined),
  q: z.string().trim().min(1).max(120).optional().catch(undefined),
  page: z.number().int().min(1).optional().catch(undefined),
});

export type AllClaimsSearch = z.infer<typeof allClaimsSearchSchema>;

export type AllClaimsFilters = Omit<AllClaimsSearch, "page">;
