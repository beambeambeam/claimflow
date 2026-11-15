import { VStack } from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";
import { useSuspenseQuery } from "@tanstack/react-query";

import { getHealthOptions } from "#/lib/api/@tanstack/react-query.gen";

export const HomePage = () => {
  const { data: health } = useSuspenseQuery(getHealthOptions());

  return (
    <VStack as="main" gap={4} padding={8}>
      <Heading level={1}>ClaimFlow</Heading>
      <Text type="body" size="lg">{`API version ${health.version}`}</Text>
    </VStack>
  );
};
