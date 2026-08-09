/**
 * Allowed commit types, following the Conventional Commits convention.
 *
 * @see {@link https://www.conventionalcommits.org/}
 */
const types = [
  {
    value: "feat",
    name: "feat:      Introduce new features",
  },
  {
    value: "fix",
    name: "fix:       Fix a bug",
  },
  {
    value: "docs",
    name: "docs:      Documentation only changes",
  },
  {
    value: "style",
    name: "style:     Code style changes (formatting, no logic change)",
  },
  {
    value: "refactor",
    name: "refactor:  Code restructuring (no feature, no bug fix)",
  },
  {
    value: "perf",
    name: "perf:      Performance improvements",
  },
  {
    value: "test",
    name: "test:      Add or update tests",
  },
  {
    value: "build",
    name: "build:     Build system or external dependency changes",
  },
  {
    value: "ci",
    name: "ci:        CI/CD configuration changes",
  },
  {
    value: "chore",
    name: "chore:     Maintenance not touching src or tests",
  },
  {
    value: "revert",
    name: "revert:    Revert a previous commit",
  },
];

/**
 * Allowed commit scopes, matching the project's top-level code areas.
 */
const scopes = [
  {
    name: "provider - Provider logic (provider/)",
    value: "provider",
  },
  {
    name: "sdk      - SDK libraries (sdk/)",
    value: "sdk",
  },
  {
    name: "examples - Example programs (examples/)",
    value: "examples",
  },
  {
    name: "tests    - Test suites (tests/)",
    value: "tests",
  },
  {
    name: "docs     - Documentation",
    value: "docs",
  },
  {
    name: "ci       - CI/CD configuration",
    value: "ci",
  },
  {
    name: "deps     - Dependency updates",
    value: "deps",
  },
  {
    name: "tooling  - Dev tooling (mise, golangci-lint, commitlint, etc.)",
    value: "tooling",
  },
];

/** @type {import('@commitlint/types').UserConfig} */
module.exports = {
  rules: {
    "body-full-stop": [0, "always", "."],
    "body-leading-blank": [2, "always"],
    "body-empty": [0, "always"],
    "body-max-length": [2, "always", "Infinity"],
    "body-max-line-length": [2, "always", 80],
    "body-min-length": [2, "always", 0],
    "body-case": [2, "always", "sentence-case"],
    // Footer is intentionally allowed — Co-authored-by trailers live here.
    "footer-leading-blank": [2, "always"],
    "footer-empty": [0, "always"],
    "footer-max-length": [2, "always", "Infinity"],
    "footer-max-line-length": [2, "always", 80],
    "footer-min-length": [2, "always", 0],
    // The header starts with a lowercase type, so header-case is disabled.
    "header-case": [0, "always"],
    "header-full-stop": [2, "never", "."],
    "header-max-length": [2, "always", 100],
    "header-min-length": [2, "always", 0],
    "header-trim": [2, "always"],
    "references-empty": [0, "never"],
    "scope-enum": [2, "always", scopes.map((scope) => scope.value)],
    "scope-case": [2, "always", "lower-case"],
    "scope-empty": [2, "never"],
    "scope-max-length": [2, "always", "Infinity"],
    "scope-min-length": [2, "always", 0],
    "subject-case": [2, "always", "sentence-case"],
    "subject-empty": [2, "never"],
    "subject-full-stop": [2, "never", "."],
    "subject-max-length": [2, "always", 100],
    "subject-min-length": [2, "always", 0],
    "subject-exclamation-mark": [0, "never"],
    "type-enum": [2, "always", types.map((type) => type.value)],
    "type-case": [2, "always", "lower-case"],
    "type-empty": [2, "never"],
    "type-max-length": [2, "always", "Infinity"],
    "type-min-length": [2, "always", 0],
  },
  parserPreset: {
    parserOpts: {
      // Matches: type(scope)!: Subject (#ref)
      headerPattern:
        /^(?<type>[a-z]+)(\((?<scope>[^()]+)\))?!?:\s(?<subject>(?:(?!#).)*(?:(?!\s).))(\s\(?(?<references>#\d*)\)?)?$/,
      breakingHeaderPattern:
        /^(?<type>[a-z]+)(\((?<scope>[^()]+)\))?!:\s(?<subject>(?:(?!#).)*(?:(?!\s).))(\s\(?(?<references>#\d*)\)?)?$/,
      headerCorrespondence: ["type", "scope", "subject", "references"],
    },
  },
  prompt: {
    allowBreakingChanges: ["feat", "fix", "refactor", "build"],
    allowCustomScopes: false,
    allowEmptyScopes: false,
    enableMultipleScopes: false,
    scopes: scopes,
    types: types,
    typesSearchValue: false,
    upperCaseSubject: true,
  },
};
