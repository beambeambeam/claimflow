// Tags use `v<generation>.<major>.<minor>.<patch>`, for example `v1.2.0.1`.
// semantic-release manages the last three numbers. The generation is fixed here; change it by hand.
const generation = 1;

export default {
  branches: ["main"],
  plugins: [
    [
      "@semantic-release/commit-analyzer",
      {
        // The angular preset ignores `type!:` headers without this pattern.
        parserOpts: { breakingHeaderPattern: "^(\\w*)(?:\\((.*)\\))?!: (.*)$" },
      },
    ],
    "@semantic-release/release-notes-generator",
    "@semantic-release/github",
  ],
  // eslint-disable-next-line no-template-curly-in-string
  tagFormat: `v${generation}.\${version}`,
};
