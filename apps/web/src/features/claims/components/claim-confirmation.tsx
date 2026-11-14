import { Badge } from "@astryxdesign/core/Badge";
import { Button } from "@astryxdesign/core/Button";
import { HStack, VStack } from "@astryxdesign/core/Layout";
import { Heading, Text } from "@astryxdesign/core/Text";

import type { Claim } from "#/lib/api/types.gen";
import { formatBaht, parseSatang } from "#/lib/format";

// What the requester sees after a claim is submitted: the claim number, the Submitted badge, and the total.
export const ClaimConfirmation = ({
  claim,
  onNewClaim,
}: {
  claim: Claim;
  onNewClaim: () => void;
}) => (
  <VStack gap={4}>
    <Heading level={1}>{`Claim ${claim.claimNo} submitted`}</Heading>
    <HStack gap={2} vAlign="center">
      <Badge label="Submitted" variant="blue" />
      <Text type="body">{`Total ${formatBaht(parseSatang(claim.total) ?? 0)}`}</Text>
    </HStack>
    <HStack gap={2}>
      <Button href="/claims" label="View my claims" variant="secondary" />
      <Button label="New claim" onClick={onNewClaim} variant="primary" />
    </HStack>
  </VStack>
);
