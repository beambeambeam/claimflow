import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { z } from "zod";

import { createClaimMutation } from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";
import { zClaimInput, zLineItemInput } from "#/lib/api/zod.gen";
import { useAppForm } from "#/lib/form";
import { formatBaht, parseSatang } from "#/lib/format";
import { isProblem } from "#/lib/problem";

import { ClaimConfirmation } from "./claim-confirmation";

// BR-05: a claim has 1 to 20 line items (F-02 §2.1).
const maxLineItems = 20;

const categoryOptions = [
  { value: "TRAVEL", label: "Travel" },
  { value: "ACCOMMODATION", label: "Accommodation" },
  { value: "MEALS", label: "Meals" },
  { value: "SUPPLIES", label: "Supplies" },
  { value: "TRAINING", label: "Training" },
  { value: "OTHER", label: "Other" },
];

const emptyItem = {
  category: "",
  description: "",
  expenseDate: "",
  amount: "",
  receiptRef: "",
};

// The shape checks of the generated schema, with the field messages of F-02 §1.4. The form keeps every value as text,
// so each field is piped into the schema of the same field.
const claimSchema = z.object({
  title: z
    .string()
    .min(1, "Enter a title (up to 120 characters).")
    .pipe(zClaimInput.shape.title),
  purpose: z
    .string()
    .min(1, "Enter a purpose (up to 1,000 characters).")
    .pipe(zClaimInput.shape.purpose),
  items: z.array(
    z.object({
      category: z
        .string()
        .min(1, "Choose a category.")
        .pipe(zLineItemInput.shape.category),
      description: z
        .string()
        .min(1, "Enter a description (up to 200 characters).")
        .pipe(zLineItemInput.shape.description),
      expenseDate: z
        .string()
        .min(1, "Enter a date as YYYY-MM-DD.")
        .pipe(zLineItemInput.shape.expenseDate),
      amount: z
        .string()
        .regex(
          /^\d{1,6}\.\d{2}$/u,
          "Enter an amount with 2 decimals, e.g. 1250.00."
        )
        .pipe(zLineItemInput.shape.amount),
      receiptRef: z
        .string()
        .min(1, "Enter the receipt reference (up to 50 characters).")
        .pipe(zLineItemInput.shape.receiptRef),
    })
  ),
});

// The sum of the amounts typed so far that are in the right format. It is display only and never sent (BR-08).
const runningTotal = (items: { amount: string }[]) =>
  items.reduce((sum, { amount }) => sum + (parseSatang(amount) ?? 0), 0);

export const ClaimForm = () => {
  const [submitted, setSubmitted] = useState<Claim | null>(null);

  const create = useMutation({
    ...createClaimMutation(),
    onSuccess: (envelope) => {
      setSubmitted(envelope.data);
    },
  });

  const form = useAppForm({
    defaultValues: { title: "", purpose: "", items: [emptyItem] },
    validators: { onSubmit: claimSchema },
    onSubmit: ({ value }) => {
      // The submit validator has checked the values, so this parse returns them in the API's types.
      create.mutate({ body: claimSchema.parse(value) });
    },
  });

  const submit = () => {
    void form.handleSubmit();
  };

  if (submitted !== null) {
    return (
      <ClaimConfirmation
        claim={submitted}
        onNewClaim={() => {
          form.reset();
          create.reset();
          setSubmitted(null);
        }}
      />
    );
  }

  const problem = isProblem(create.error) ? create.error : null;

  return (
    <VStack
      as="form"
      gap={4}
      hAlign="stretch"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <Heading level={1}>New Claim</Heading>

      <form.AppField name="title">
        {(field) => <field.TextField label="Title" />}
      </form.AppField>

      <form.AppField name="purpose">
        {(field) => <field.TextAreaField label="Purpose" />}
      </form.AppField>

      <Heading level={2}>Line items</Heading>

      <form.Field mode="array" name="items">
        {(items) => (
          <VStack gap={4} hAlign="stretch">
            {items.state.value.map((_, index) => (
              // Rows have no id of their own, and removing one re-reads every value by position.
              // oxlint-disable-next-line react/no-array-index-key
              <HStack gap={2} key={index} vAlign="start" wrap="wrap">
                <form.AppField name={`items[${index}].category`}>
                  {(field) => (
                    <field.SelectField
                      label="Category"
                      options={categoryOptions}
                      placeholder="Choose a category"
                    />
                  )}
                </form.AppField>
                <form.AppField name={`items[${index}].description`}>
                  {(field) => <field.TextField label="Description" />}
                </form.AppField>
                <form.AppField name={`items[${index}].expenseDate`}>
                  {(field) => <field.DateField label="Expense date" />}
                </form.AppField>
                <form.AppField name={`items[${index}].amount`}>
                  {(field) => (
                    <field.TextField
                      inputMode="decimal"
                      label="Amount"
                      placeholder="1250.00"
                    />
                  )}
                </form.AppField>
                <form.AppField name={`items[${index}].receiptRef`}>
                  {(field) => <field.TextField label="Receipt reference" />}
                </form.AppField>
                <Button
                  isDisabled={items.state.value.length === 1}
                  label="Remove"
                  onClick={() => {
                    items.removeValue(index);
                  }}
                  variant="ghost"
                />
              </HStack>
            ))}
            <HStack>
              <Button
                isDisabled={items.state.value.length >= maxLineItems}
                label="Add line item"
                onClick={() => {
                  items.pushValue(emptyItem);
                }}
              />
            </HStack>
          </VStack>
        )}
      </form.Field>

      <form.Subscribe selector={(state) => runningTotal(state.values.items)}>
        {(total) => (
          <Text type="body" weight="bold">{`Total ${formatBaht(total)}`}</Text>
        )}
      </form.Subscribe>

      {create.isError ? (
        <Banner
          description={
            problem === null ? undefined : `Request ID: ${problem.requestId}`
          }
          endContent={<Button label="Try again" onClick={submit} size="sm" />}
          status="error"
          title="Something went wrong."
        />
      ) : null}

      <HStack>
        <Button
          isLoading={create.isPending}
          label="Submit"
          type="submit"
          variant="primary"
        />
      </HStack>
    </VStack>
  );
};
