export default {
  branches: ["main"],
  // eslint-disable-next-line no-template-curly-in-string
  tagFormat: "v${version}",
  plugins: [
    [
      "@semantic-release/commit-analyzer",
      {
        // The angular preset ignores `type!:` headers without this pattern.
        parserOpts: { breakingHeaderPattern: "^(\\w*)(?:\\((.*)\\))?!: (.*)$" },
      },
    ],
    "@semantic-release/release-notes-generator",
    ["@semantic-release/npm", { npmPublish: false }],
    [
      "@semantic-release/git",
      {
        assets: ["package.json"],
        // eslint-disable-next-line no-template-curly-in-string
        message: "chore: release v${nextRelease.version} [skip ci]",
      },
    ],
    ["@semantic-release/gitlab", { useJobToken: true }],
  ],
};
