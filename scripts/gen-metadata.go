//go:build ignore

// Go SDK docs metadata generator. AAASM-4311 / Epic AAASM-4309.
//
// Reads authoritative values from the repo root:
//
//   - go.mod           — module path (source of truth for Go import path)
//   - VERSION          — gateway protocol version pinned by this SDK
//   - metadata/sdk.yaml — docs-facing values (canonical URLs, Go floor, install
//     command template)
//
// Writes three artifacts:
//
//  1. internal/version/metadata.go — a generated Go source file exposing the
//     minimal set of constants actually consumed by code (module path,
//     protocol version, docs / repo URLs). Header starts with the canonical
//     `// Code generated ... DO NOT EDIT.` line so gofmt/goimports and human
//     reviewers recognise it as generated.
//
//  2. assembly/version.go — the public `assembly.Version` constant, sourced
//     from the same VERSION file that feeds `internal/version.ProtocolVersion`.
//     AAASM-4326 replaced the previously hand-maintained literal so the public
//     API surface can no longer drift from the shared metadata source.
//
//  3. README.md — rewrites two bounded blocks, preserving every byte of the
//     surrounding hand-written prose:
//     - <!-- BEGIN GENERATED: sdk-metadata --> ... <!-- END GENERATED: sdk-metadata -->
//     (the "Metadata at a glance" table), and
//     - <!-- BEGIN GENERATED: protocol-version --> ... <!-- END GENERATED: protocol-version -->
//     (the inline protocol-version literal in the "Project status" prose).
//     AAASM-4920 brought this literal under the generator so the README can
//     no longer restate — and drift from — the VERSION-file SoT (ADR 0013).
//
// The program has zero network access and reads only local repo state.
//
// Invocation (from repo root):
//
//	go run scripts/gen-metadata.go
//
// The build tag `//go:build ignore` keeps this file out of `go build ./...`
// so it never links into the SDK module — the standard pattern for
// `go generate`-invoked scripts.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
)

// sharedMetadata mirrors metadata/sdk.yaml. Fields are documented at the YAML
// source; only add a field here after adding it in the YAML.
type sharedMetadata struct {
	DocsURL        string `yaml:"docsUrl"`
	RepoURL        string `yaml:"repoUrl"`
	ReleasesURL    string `yaml:"releasesUrl"`
	GoMinVersion   string `yaml:"goMinVersion"`
	InstallCommand string `yaml:"installCommand"`
}

// resolvedMetadata is the flattened view handed to the generators.
type resolvedMetadata struct {
	ModulePath      string
	ProtocolVersion string
	DocsURL         string
	RepoURL         string
	ReleasesURL     string
	GoMinVersion    string
	InstallCommand  string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("gen-metadata: ")

	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}

	meta, err := load(repoRoot)
	if err != nil {
		log.Fatalf("load metadata: %v", err)
	}

	if err := writeVersionGo(repoRoot, meta); err != nil {
		log.Fatalf("write internal/version/metadata.go: %v", err)
	}

	if err := writeAssemblyVersion(repoRoot, meta); err != nil {
		log.Fatalf("write assembly/version.go: %v", err)
	}

	if err := rewriteReadmeBlock(repoRoot, meta); err != nil {
		log.Fatalf("rewrite README.md sdk-metadata block: %v", err)
	}
}

// findRepoRoot walks up from the current working directory until it finds a
// go.mod file. This lets the generator be invoked either directly from the
// repo root (`go run scripts/gen-metadata.go`) or by `go generate` running
// inside a nested package directory.
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found in %s or any parent", cwd)
		}
		dir = parent
	}
}

func load(repoRoot string) (resolvedMetadata, error) {
	modBytes, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return resolvedMetadata{}, fmt.Errorf("read go.mod: %w", err)
	}
	mf, err := modfile.Parse("go.mod", modBytes, nil)
	if err != nil {
		return resolvedMetadata{}, fmt.Errorf("parse go.mod: %w", err)
	}
	if mf.Module == nil || mf.Module.Mod.Path == "" {
		return resolvedMetadata{}, fmt.Errorf("go.mod has no module path")
	}

	versionBytes, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		return resolvedMetadata{}, fmt.Errorf("read VERSION: %w", err)
	}
	protocolVersion := strings.TrimSpace(string(versionBytes))
	if protocolVersion == "" {
		return resolvedMetadata{}, fmt.Errorf("VERSION file is empty")
	}

	yamlBytes, err := os.ReadFile(filepath.Join(repoRoot, "metadata", "sdk.yaml"))
	if err != nil {
		return resolvedMetadata{}, fmt.Errorf("read metadata/sdk.yaml: %w", err)
	}
	var shared sharedMetadata
	if err := yaml.Unmarshal(yamlBytes, &shared); err != nil {
		return resolvedMetadata{}, fmt.Errorf("parse metadata/sdk.yaml: %w", err)
	}

	requireNonEmpty := map[string]string{
		"docsUrl":        shared.DocsURL,
		"repoUrl":        shared.RepoURL,
		"releasesUrl":    shared.ReleasesURL,
		"goMinVersion":   shared.GoMinVersion,
		"installCommand": shared.InstallCommand,
	}
	for name, value := range requireNonEmpty {
		if strings.TrimSpace(value) == "" {
			return resolvedMetadata{}, fmt.Errorf("metadata/sdk.yaml: %s is required", name)
		}
	}

	installCommand := strings.ReplaceAll(shared.InstallCommand, "{module}", mf.Module.Mod.Path)

	return resolvedMetadata{
		ModulePath:      mf.Module.Mod.Path,
		ProtocolVersion: protocolVersion,
		DocsURL:         shared.DocsURL,
		RepoURL:         shared.RepoURL,
		ReleasesURL:     shared.ReleasesURL,
		GoMinVersion:    shared.GoMinVersion,
		InstallCommand:  installCommand,
	}, nil
}

// versionTemplate renders internal/version/metadata.go. Keep the generated set
// minimal — every exported constant is a maintenance surface, so only add one
// after there is a real Go consumer for it.
const versionTemplate = `// Code generated by scripts/gen-metadata.go. DO NOT EDIT.
//
// To regenerate, run ` + "`go generate ./...`" + ` from the repo root. See
// metadata/sdk.yaml and go.mod for the authoritative source values.

package version

// ModulePath is the canonical Go import path for this SDK, sourced from
// go.mod. Consumers that need to construct a self-referential URL (e.g. an
// error message pointing users at the repo) should read this constant
// instead of hard-coding the string.
const ModulePath = %q

// ProtocolVersion is the gateway wire-protocol version this SDK build is
// pinned to, sourced from the top-level VERSION file. It is the value the
// gateway is expected to advertise on Init handshakes.
const ProtocolVersion = %q

// DocsURL is the canonical documentation site for the Go SDK.
const DocsURL = %q

// RepoURL is the canonical GitHub repository URL for the Go SDK.
const RepoURL = %q
`

func writeVersionGo(repoRoot string, meta resolvedMetadata) error {
	target := filepath.Join(repoRoot, "internal", "version", "metadata.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir internal/version: %w", err)
	}
	content := fmt.Sprintf(
		versionTemplate,
		meta.ModulePath,
		meta.ProtocolVersion,
		meta.DocsURL,
		meta.RepoURL,
	)
	return os.WriteFile(target, []byte(content), 0o644)
}

// assemblyVersionTemplate renders assembly/version.go — the public
// `assembly.Version` constant. The package's godoc lives in assembly/doc.go, so
// this generated file intentionally omits a package-doc comment to avoid a
// second package-level block. The const-level godoc is preserved verbatim from
// the previously hand-maintained file so `go doc github.com/ai-agent-assembly/go-sdk/assembly.Version`
// still returns the same prose.
const assemblyVersionTemplate = `// Code generated by scripts/gen-metadata.go. DO NOT EDIT.
//
// To regenerate, run ` + "`go generate ./...`" + ` from the repo root. The value
// is sourced from the top-level VERSION file (same source as
// internal/version.ProtocolVersion) — edit VERSION, not this file.

package assembly

// Version is the published version of the agent-assembly Go SDK.
//
// Pre-release values follow the SemVer pre-release identifier form
// (` + "`MAJOR.MINOR.PATCH-alpha.N`" + `). The git tag mirrors this with a
// leading ` + "`v`" + ` (e.g. ` + "`v0.0.1-alpha.1`" + `).
const Version = %q
`

func writeAssemblyVersion(repoRoot string, meta resolvedMetadata) error {
	target := filepath.Join(repoRoot, "assembly", "version.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir assembly: %w", err)
	}
	content := fmt.Sprintf(assemblyVersionTemplate, meta.ProtocolVersion)
	return os.WriteFile(target, []byte(content), 0o644)
}

// readmeBlockRE matches the entire bounded block (including sentinels) so the
// generator replaces it in-place without touching surrounding prose.
var readmeBlockRE = regexp.MustCompile(
	`(?s)<!--\s*BEGIN GENERATED: sdk-metadata\s*-->.*?<!--\s*END GENERATED: sdk-metadata\s*-->`,
)

func renderReadmeBlock(meta resolvedMetadata) string {
	var b strings.Builder
	b.WriteString("<!-- BEGIN GENERATED: sdk-metadata -->\n")
	b.WriteString("<!-- GENERATED BY scripts/gen-metadata.go — DO NOT EDIT. -->\n")
	b.WriteString("\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("| --- | --- |\n")
	fmt.Fprintf(&b, "| Module | `%s` |\n", meta.ModulePath)
	fmt.Fprintf(&b, "| Protocol version | `%s` |\n", meta.ProtocolVersion)
	fmt.Fprintf(&b, "| Go floor | `>= %s` |\n", meta.GoMinVersion)
	fmt.Fprintf(&b, "| Docs | <%s> |\n", meta.DocsURL)
	fmt.Fprintf(&b, "| Releases | <%s> |\n", meta.ReleasesURL)
	b.WriteString("\n")
	b.WriteString("Install:\n\n")
	b.WriteString("```bash\n")
	b.WriteString(meta.InstallCommand)
	b.WriteString("\n```\n")
	b.WriteString("<!-- END GENERATED: sdk-metadata -->")
	return b.String()
}

// readmeProtocolVersionRE matches the inline protocol-version block embedded in
// the "Project status" prose. It is a separate, single-line sentinel pair (no
// surrounding newlines) so the generated literal can sit mid-sentence — the
// prose narrates the value, this block owns it. Bringing it under the generator
// closes the ADR 0013 gap where the README restated the VERSION-file literal in
// hand-maintained prose and could drift from the SoT (AAASM-4920).
var readmeProtocolVersionRE = regexp.MustCompile(
	`(?s)<!--\s*BEGIN GENERATED: protocol-version\s*-->.*?<!--\s*END GENERATED: protocol-version\s*-->`,
)

// renderProtocolVersionBlock renders the inline protocol-version sentinel pair.
// It carries no DO-NOT-EDIT banner because an inline HTML comment mid-sentence
// would render as visible prose noise; the sentinel names alone mark it as
// generated, and CONTRIBUTING.md documents the pipeline.
func renderProtocolVersionBlock(meta resolvedMetadata) string {
	return fmt.Sprintf(
		"<!-- BEGIN GENERATED: protocol-version -->`%s`<!-- END GENERATED: protocol-version -->",
		meta.ProtocolVersion,
	)
}

func rewriteReadmeBlock(repoRoot string, meta resolvedMetadata) error {
	target := filepath.Join(repoRoot, "README.md")
	current, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read README.md: %w", err)
	}
	if !readmeBlockRE.Match(current) {
		return fmt.Errorf(
			"README.md is missing the sdk-metadata generated block; " +
				"add a `<!-- BEGIN GENERATED: sdk-metadata -->` / " +
				"`<!-- END GENERATED: sdk-metadata -->` pair to opt in",
		)
	}
	if !readmeProtocolVersionRE.Match(current) {
		return fmt.Errorf(
			"README.md is missing the protocol-version generated block; " +
				"add a `<!-- BEGIN GENERATED: protocol-version -->` / " +
				"`<!-- END GENERATED: protocol-version -->` pair to opt in",
		)
	}
	updated := readmeBlockRE.ReplaceAllLiteral(current, []byte(renderReadmeBlock(meta)))
	updated = readmeProtocolVersionRE.ReplaceAllLiteral(updated, []byte(renderProtocolVersionBlock(meta)))
	if bytes.Equal(current, updated) {
		return nil
	}
	return os.WriteFile(target, updated, 0o644)
}
