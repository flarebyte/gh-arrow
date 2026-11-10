package flyb

source: "gh-arrow-design"
name: "gh-arrow"
modules: ["core"]

reports: [{
 title: "gh-arrow specification"
 filepath: "../design/specification.md"
 description: "Generated from doc/design-meta; edit the source and regenerate."
 sections: [
  {title: "Overview", sections: [{title: "Overview details", notes: ["arrow.overview", "arrow.decisions", "arrow.references"]}]},
  {title: "Interface", sections: [{title: "Interface details", notes: ["arrow.commands", "arrow.usecases", "arrow.config", "arrow.listing"]}]},
  {title: "Requirements", sections: [{title: "Requirements details", notes: ["arrow.rules", "arrow.constraints", "arrow.acceptance"]}]},
  {title: "Navigation and follow-up", sections: [{title: "Navigation and follow-up details", notes: ["arrow.concepts", "arrow.open-questions"]}]},
 ]
}]

notes: [
 {name: "arrow.acceptance", title: "Acceptance scenarios", filepath: "examples/acceptance.csv", arguments: ["format-csv=table"]},
 {name: "arrow.commands", title: "Command surface", filepath: "examples/commands.csv", arguments: ["format-csv=table"]},
 {name: "arrow.concepts", title: "Concept registry", filepath: "examples/concepts.csv", arguments: ["format-csv=table"]},
 {name: "arrow.config", title: "Mapping configuration", filepath: "examples/config.cue"},
 {name: "arrow.constraints", title: "Scope limits", filepath: "examples/constraints.csv", arguments: ["format-csv=table"]},
 {name: "arrow.decisions", title: "Design decisions", filepath: "examples/decisions.csv", arguments: ["format-csv=table"]},
 {name: "arrow.listing", title: "JSON listing contract", filepath: "examples/listing.ts"},
 {name: "arrow.open-questions", title: "Open questions", filepath: "examples/open-questions.csv", arguments: ["format-csv=table"]},
 {name: "arrow.overview", title: "Overview", markdown: "gh-arrow is a Go extension invoked as `gh arrow`. It copies, displays and lists files from GitHub or mapped local working trees, and discovers files across organizations or users. File commands and discovery are implemented with offline acceptance tests and native CI verification; no release has been published. The original proposal is preserved in `doc/history/initial-spec.md`; decisions below explicitly replace its standalone name and public-only remote authentication restrictions."},
 {name: "arrow.references", title: "GitHub extension references", markdown: "Packaging follows [Creating GitHub CLI extensions](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions) and the [extension command manual](https://cli.github.com/manual/gh_extension). The repository target name is `gh-arrow`, the executable is `gh-arrow`, and the invocation is `gh arrow`. The hosted repository is now `gh-arrow`; release publication remains a future implementation task."},
 {name: "arrow.rules", title: "Behavioral rules", filepath: "examples/rules.csv", arguments: ["format-csv=table"]},
 {name: "arrow.usecases", title: "Usage examples", filepath: "examples/usecases.csv", arguments: ["format-csv=table"]},
]

relationships: []
argumentRegistry: {version: "1", arguments: [{name: "format-csv", valueType: "enum", scopes: ["note"], allowedValues: ["table", "raw"]}]}
