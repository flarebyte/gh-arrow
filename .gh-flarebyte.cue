package ghflarebyte

project: {
	org:  "flarebyte"
	repo: "gh-arrow"
}

sync: {
	mode: "push"
}

repository: {
	description:   "Hit your GitHub target every time with this utility to copy files"
	defaultBranch: "main"
	homepage:      "https://github.com/flarebyte/gh-arrow"
	visibility:    "public"
	template:      false
	topics: [
		"gh-extension",
		"github-cli",
		"git",
		"flarebyte",
	]
	labels: [
		{
			name:        "bug"
			color:       "B60205"
			description: "Something is broken"
		},
		{
			name:        "enhancement"
			color:       "0E8A16"
			description: "New feature"
		},
	]
	features: {
		issues:                        true
		wiki:                          false
		projects:                      false
		discussions:                   false
		autoMerge:                     true
		mergeCommit:                   false
		rebaseMerge:                   false
		squashMerge:                   true
		squashMergeCommitMessage:      "pr-title"
		deleteBranchOnMerge:           true
		allowForking:                  false
		allowUpdateBranch:             false
		advancedSecurity:              true
		secretScanning:                true
		secretScanningPushProtection:  true
	}
}

build: {
	language:     "go"
	mode:         "binary"
	mainPackage:  "./cmd/gh-arrow"
	outputDir:    "build"
	checksumFile: "build/checksums.txt"
	artifactTargetSuffix: true
	targets: [
		"linux-amd64",
		"darwin-arm64",
		"windows-amd64",
	]
}

go: {
	cacheDir:    "./.gocache"
	modCacheDir: "./.gomodcache"
	toolchain:   "local"
	cgo: enabled: false
}

devOutput: {
	color:      "auto"
	style:      "summary"
	showPassed: true
}

release: {
	versionSource:    "main.project.yaml"
	tagPrefix:        "v"
	notesMode:        "generate-notes"
	includeArtifacts: true
	artifactDir:      "build"
	includeChecksums: true
}
