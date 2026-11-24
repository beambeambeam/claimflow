import { z } from "zod";

// The contract keeps the message of each field error and the BR-04 comment limits as schema extensions. The generator
// registers them on the generated schemas (see `openapi-ts.config.ts`), so the web and the API read one source.
// The API's message for a field that has no `x-message` in the contract.
const defaultMessage = "Enter a valid value.";

// The registered metadata of a generated schema. The registry is typed loosely, so it is parsed here.
const contractMetadata = z.object({
  message: z.string().default(defaultMessage),
  minLength: z.number().optional(),
  maxLength: z.number().optional(),
});

const metadata = (schema: z.ZodType) =>
  contractMetadata.parse(z.globalRegistry.get(schema) ?? {});

// The `x-message` of a generated schema: the text of a field error on it.
export const contractMessage = (schema: z.ZodType): string =>
  metadata(schema).message;

// The required `x-min-length` of a generated schema that supplies a minimum to the form.
export const contractMinLength = (schema: z.ZodType): number =>
  z.number().parse(metadata(schema).minLength);

// The `x-max-length` of a generated schema, or undefined when the contract gives none.
export const contractMaxLength = (schema: z.ZodType): number | undefined =>
  metadata(schema).maxLength;

// A form field kept as text and checked by the generated schema of the same field. Any failure gets the contract's
// one message, and the parse returns the value in the API's type.
export const contractField = <T extends z.ZodType<unknown, string>>(
  schema: T
) =>
  z
    .string()
    .refine((value) => schema.safeParse(value).success, {
      abort: true,
      message: contractMessage(schema),
    })
    .pipe(schema);
