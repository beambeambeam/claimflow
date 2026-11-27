import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";
import { Grid } from "@astryxdesign/core/Grid";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Link } from "@astryxdesign/core/Link";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useState } from "react";
import { z } from "zod";

import type { ClaimInput } from "#/lib/api/types.gen";
import { zClaimInput, zLineItemInput } from "#/lib/api/zod.gen";
import { contractField } from "#/lib/contract";
import { useAppForm } from "#/lib/form";
import { formatBaht, parseSatang } from "#/lib/format";
import { categoryLabels } from "#/lib/labels";
import { isProblem, problemFieldErrors } from "#/lib/problem";

// BR-05: a claim has 1 to 20 line items (F-02 §2.1).
const maxLineItems = 20;

const categoryOptions = Object.entries(categoryLabels).map(
  ([value, label]) => ({ value, label })
);

const emptyItem = {
  category: "",
  description: "",
  expenseDate: "",
  amount: "",
  receiptRef: "",
};

// The shape checks of the generated schema, with the contract's field messages (F-02 §1.4). The form keeps every value
// as text, so each field is checked by the schema of the same field.
const claimSchema = z.object({
  title: contractField(zClaimInput.shape.title),
  purpose: contractField(zClaimInput.shape.purpose),
  items: z.array(
    z.object({
      category: contractField(zLineItemInput.shape.category),
      description: contractField(zLineItemInput.shape.description),
      expenseDate: contractField(zLineItemInput.shape.expenseDate),
      amount: contractField(zLineItemInput.shape.amount),
      receiptRef: contractField(zLineItemInput.shape.receiptRef),
    })
  ),
});

// The sum of the amounts typed so far that are in the right format. It is display only and never sent (BR-08).
const runningTotal = (items: { amount: string }[]) =>
  items.reduce((sum, { amount }) => sum + (parseSatang(amount) ?? 0), 0);

type SubmitIntent = "primary" | "draft";

interface ClaimFormProps {
  initialValues?: ClaimInput;
  cancelHref?: string;
  error: unknown;
  onSave: (body: ClaimInput, intent: SubmitIntent) => Promise<void>;
}

export const ClaimForm = ({
  initialValues,
  cancelHref,
  error,
  onSave,
}: ClaimFormProps) => {
  const [intent, setIntent] = useState<SubmitIntent>("primary");
  const editing = cancelHref !== undefined;

  const form = useAppForm({
    defaultValues: initialValues ?? {
      title: "",
      purpose: "",
      items: [emptyItem],
    },
    validators: { onSubmit: claimSchema },
    onSubmitMeta: { intent },
    // Awaiting the request keeps `isSubmitting` true until it settles, so `submit` ignores clicks in the meantime.
    onSubmit: async ({ value, meta }) => {
      try {
        // The submit validator has checked the values, so this parse returns them in the API's types.
        await onSave(claimSchema.parse(value), meta.intent);
      } catch (error_) {
        if (isProblem(error_)) {
          // Each server error goes under its input, as `items[1].expenseDate`; the values stay as typed.
          form.setErrorMap({
            onSubmit: { fields: problemFieldErrors(error_) },
          });
        }
      }
    },
  });

  const submit = (nextIntent: SubmitIntent) => {
    // F02-AC-13: a second click lands before the busy Submit renders. `handleSubmit` sets `isSubmitting` before it
    // awaits anything, so this check stops a double-click from sending the claim twice.
    if (form.state.isSubmitting) {
      return;
    }

    setIntent(nextIntent);

    // Earlier server errors would keep the form from submitting; the server answers again for these values.
    form.setErrorMap({ onSubmit: { fields: {} } });
    void form.handleSubmit({ intent: nextIntent });
  };

  const problem = isProblem(error) ? error : null;
  const fieldErrors = problem === null ? {} : problemFieldErrors(problem);

  // `items` (the count) and `total` have no input of their own, so their messages show above Submit.
  const claimErrors = ["items", "total"].flatMap((name) =>
    name in fieldErrors ? [fieldErrors[name].message] : []
  );

  const hasFieldErrors = Object.keys(fieldErrors).length > 0;

  return (
    <VStack
      as="form"
      gap={4}
      hAlign="stretch"
      onSubmit={(event) => {
        event.preventDefault();
        submit("primary");
      }}
    >
      <Heading level={1}>{editing ? "Edit Claim" : "New Claim"}</Heading>

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
              // Rows have no id of their own, and removing one re-reads every value by position. Each field has a grid
              // track of its own, so an error under one field does not move the others.
              // oxlint-disable-next-line react/no-array-index-key
              <Grid
                align="start"
                columns={{ minWidth: 160, max: 6 }}
                gap={2}
                key={index}
              >
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
                <HStack>
                  <Button
                    isDisabled={items.state.value.length === 1}
                    label="Remove"
                    onClick={() => {
                      items.removeValue(index);
                    }}
                    variant="ghost"
                  />
                </HStack>
              </Grid>
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

      {claimErrors.map((message) => (
        <Banner key={message} status="error" title={message} />
      ))}

      {error !== null && !hasFieldErrors ? (
        <Banner
          description={
            problem === null ? undefined : `Request ID: ${problem.requestId}`
          }
          endContent={
            <Button
              label="Try again"
              onClick={() => {
                submit(intent);
              }}
              size="sm"
            />
          }
          status="error"
          title="Something went wrong."
        />
      ) : null}

      <form.Subscribe selector={(state) => state.isSubmitting}>
        {(isSubmitting) => (
          <HStack>
            <Button
              isDisabled={isSubmitting}
              isLoading={isSubmitting && intent === "primary"}
              label={editing ? "Save changes" : "Submit"}
              type="submit"
              variant="primary"
            />
            {editing ? (
              <Link href={cancelHref}>Cancel</Link>
            ) : (
              <Button
                isDisabled={isSubmitting}
                isLoading={isSubmitting && intent === "draft"}
                label="Save as draft"
                onClick={() => {
                  submit("draft");
                }}
                variant="secondary"
              />
            )}
          </HStack>
        )}
      </form.Subscribe>
    </VStack>
  );
};
