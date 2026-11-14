import { VStack } from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";

// The page for a member without the claim:create permission, such as a Finance Officer (BR-15). It shows no form.
export const CannotCreateClaim = () => (
  <VStack gap={2}>
    <Heading level={1}>You can&apos;t create claims.</Heading>
    <Text type="body" color="secondary">
      Your account has no assigned reviewer, so it cannot submit claims.
    </Text>
  </VStack>
);
