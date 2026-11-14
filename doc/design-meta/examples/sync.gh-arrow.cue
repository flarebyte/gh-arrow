// Project file mappings; destinations are relative to this manifest.
// Sources use mapped working trees or the remote default branch.
version: 1
files: [
    {
        source: "github:owner:project:path/to/file.yaml"
        destination: "local.yaml"
    },
    {
        source: "github:owner:standards:configs/lint.yaml"
        destination: ".config/lint.yaml"
    },
]
