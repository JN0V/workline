package linters

import "slices"

// ecosystem are the tools a language's own toolchain brings, which
// trunk's table leaves out (ADR-0049, section 2): suggested before it,
// and counted where a project runs them. Kept by hand, and short: a
// tool of a third party belongs in trunk's table.
var ecosystem = []Tool{
	{Name: "go vet", Reads: map[string][]string{Lint: {"go"}, Types: {"go"}}, Suggest: "files", SeenAs: []string{"go vet"}, Run: "go vet ./..."},
	{Name: "go build", Reads: map[string][]string{Types: {"go"}}, Suggest: "never", SeenAs: []string{"go build"}},
	{Name: "go test", Reads: map[string][]string{Types: {"go"}}, Suggest: "never", SeenAs: []string{"go test"}},
	{Name: "cargo check", Reads: map[string][]string{Types: {"rust"}}, Suggest: "files", SeenAs: []string{"cargo check"}, Run: "cargo check"},
	{Name: "cargo build", Reads: map[string][]string{Types: {"rust"}}, Suggest: "never", SeenAs: []string{"cargo build"}},
	{Name: "cargo test", Reads: map[string][]string{Types: {"rust"}}, Suggest: "never", SeenAs: []string{"cargo test"}},
	{Name: "tsc", Reads: map[string][]string{Types: {"typescript"}}, Suggest: "config", Configs: []string{"tsconfig.json"}, SeenAs: []string{"tsc", "vue-tsc"}, Run: "tsc --noEmit"},
	// PlatformIO's own check runs cppcheck by default; it needs a build
	// that works, which the project's CI shows.
	{Name: "pio check", Reads: map[string][]string{Lint: {"c/c++"}}, Suggest: "config", Configs: []string{"platformio.ini"}, SeenAs: []string{"pio check", "platformio check"}, Run: "pio check"},
	{Name: "pio run", Reads: map[string][]string{Types: {"c/c++"}}, Suggest: "never", SeenAs: []string{"pio run", "platformio run"}},
	{Name: "pio test", Reads: map[string][]string{Types: {"c/c++"}}, Suggest: "never", SeenAs: []string{"pio test", "platformio test"}},
	// Not in trunk's table, though C and C++ projects run it.
	{Name: "cppcheck", Reads: map[string][]string{Lint: {"c/c++"}}, Suggest: "never", SeenAs: []string{"cppcheck"}},
}

// ecosystemTypes are infrastructure files trunk's table has no type for:
// a Kubernetes manifest, told by its keys, and a Helm chart.
var ecosystemTypes = []fileType{
	{Name: "kubernetes", Extensions: []string{"yaml", "yml"}, Keys: []string{"apiVersion", "kind"}},
	{Name: "helm", Filenames: []string{"Chart.yaml"}},
}

// infrastructure are the file types an infrastructure scanner reads, in
// place of every YAML and JSON file trunk gives it: suggested where they
// are, never for a plain config (ADR-0049, section 8c).
var infrastructure = []string{"cloudformation", "docker", "helm", "kubernetes", "terraform"}

// correct sets the kinds trunk's table cannot say, having none for
// types: a type checker reads types, not lint; a linter that loads the
// code typed reads both. Brakeman, a security scanner, trunk files as a
// linter; the infrastructure scanners read infrastructure only.
func correct(x Tool) Tool {
	reads := map[string][]string{}
	for k, v := range x.Reads {
		reads[k] = slices.Clone(v)
	}
	switch x.Name {
	case "checkov", "trivy":
		reads[Security] = slices.Clone(infrastructure)
	case "kube-linter":
		reads[Lint] = []string{"kubernetes"}
	case "mypy", "pyright", "ty":
		reads[Types] = reads[Lint]
		delete(reads, Lint)
	case "brakeman":
		reads[Security] = reads[Lint]
		delete(reads, Lint)
	case "golangci-lint", "clippy":
		reads[Types] = slices.Clone(reads[Lint])
	}
	x.Reads = reads
	return x
}
