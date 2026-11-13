import { VStack } from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";

// Page body for routes whose feature is not built yet.
export const PlaceholderPage = ({
  title,
  note,
}: {
  title: string;
  note: string;
}) => (
  <VStack gap={2}>
    <Heading level={1}>{title}</Heading>
    <Text type="body" color="secondary">
      {note}
    </Text>
  </VStack>
);
