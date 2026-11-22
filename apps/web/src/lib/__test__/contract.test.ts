import { describe, expect, it } from "vite-plus/test";
import { z } from "zod";

import { zClaimDecision, zClaimInput, zMoney } from "#/lib/api/zod.gen";
import {
  contractField,
  contractMaxLength,
  contractMessage,
} from "#/lib/contract";

describe("contract metadata", () => {
  it("reads the message and the comment limit of the contract", () => {
    expect(contractMessage(zClaimInput.shape.title)).toBe(
      "Enter a title (up to 120 characters)."
    );
    expect(contractMaxLength(zClaimDecision.shape.comment.unwrap())).toBe(500);
  });

  it("falls back when the contract gives no extension", () => {
    expect(contractMessage(z.string())).toBe("Enter a valid value.");
    expect(contractMaxLength(z.string())).toBeUndefined();
  });

  it("gives one contract message for any failure of a field", () => {
    const amount = contractField(zMoney);

    expect(amount.parse("1250.00")).toBe("1250.00");
    expect(amount.safeParse("12.5").error?.issues).toMatchObject([
      { message: "Enter an amount with 2 decimals, e.g. 1250.00." },
    ]);
  });
});
