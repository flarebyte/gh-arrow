// Specification examples only; this project is implemented in Go.
// --json always emits an array; mode determines the record shape.
export type SourceListing = { path: string }; // relative to matched directory prefix
export type DiscoveryListing = { repository: string; path: string }; // repo-root-relative
export const sourceExample: SourceListing[] = [{ path: "guide.md" }];
export const discoveryExample: DiscoveryListing[] = [
  { repository: "tools-org/example", path: "README.md" },
];
