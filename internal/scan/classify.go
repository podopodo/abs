package scan

import (
	"path/filepath"
	"strings"
)

var extensionKind = map[string]string{
	".go": "go", ".py": "python", ".js": "javascript", ".jsx": "javascript", ".mjs": "javascript", ".cjs": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".java": "java", ".kt": "kotlin", ".kts": "kotlin", ".cs": "csharp",
	".php": "php", ".rb": "ruby", ".rs": "rust", ".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".hpp": "cpp",
	".swift": "swift", ".scala": "scala", ".dart": "dart", ".lua": "lua", ".ex": "elixir", ".exs": "elixir",
	".html": "html", ".htm": "html", ".xhtml": "html", ".vue": "template", ".svelte": "template", ".astro": "template",
	".twig": "template", ".hbs": "template", ".handlebars": "template", ".ejs": "template", ".jinja": "template", ".jinja2": "template",
	".css": "css", ".scss": "css", ".sass": "css", ".less": "css", ".styl": "css",
	".sh": "shell", ".bash": "shell", ".zsh": "shell", ".fish": "shell", ".ps1": "powershell", ".bat": "batch", ".cmd": "batch",
	".sql": "sql", ".graphql": "graphql", ".gql": "graphql", ".proto": "protobuf",
	".json": "config", ".jsonc": "config", ".yaml": "config", ".yml": "config", ".toml": "config", ".ini": "config", ".conf": "config", ".env": "config", ".xml": "config",
	".tf": "terraform", ".tfvars": "terraform", ".hcl": "terraform", ".dockerfile": "docker",
	".md": "docs", ".rst": "docs", ".adoc": "docs",
}

var knownNames = map[string]string{
	"dockerfile": "docker", "containerfile": "docker", "makefile": "make", "gnumakefile": "make", "jenkinsfile": "ci",
	"procfile": "config", "vagrantfile": "ruby", "gemfile": "ruby", "rakefile": "ruby", "justfile": "make",
	"nginx.conf": "nginx", "apache2.conf": "apache", "httpd.conf": "apache", "compose.yaml": "config", "compose.yml": "config",
	"docker-compose.yml": "config", "docker-compose.yaml": "config", "package.json": "config", "go.mod": "config", "go.work": "config",
	"cargo.toml": "config", "pom.xml": "config", "build.gradle": "config", "build.gradle.kts": "config",
}

func classify(path string, text string) string {
	base := strings.ToLower(filepath.Base(path))
	if k, ok := knownNames[base]; ok {
		return k
	}
	norm := filepath.ToSlash(strings.ToLower(path))
	if strings.Contains(norm, "/.github/workflows/") || strings.HasPrefix(norm, ".github/workflows/") {
		return "workflow"
	}
	if k, ok := extensionKind[strings.ToLower(filepath.Ext(base))]; ok {
		return k
	}
	first := text
	if i := strings.IndexByte(first, '\n'); i >= 0 {
		first = first[:i]
	}
	if strings.HasPrefix(first, "#!") {
		f := strings.ToLower(first)
		switch {
		case strings.Contains(f, "bash"), strings.Contains(f, "/sh"), strings.Contains(f, "zsh"):
			return "shell"
		case strings.Contains(f, "python"):
			return "python"
		case strings.Contains(f, "node"), strings.Contains(f, "deno"):
			return "javascript"
		case strings.Contains(f, "ruby"):
			return "ruby"
		case strings.Contains(f, "php"):
			return "php"
		case strings.Contains(f, "pwsh"), strings.Contains(f, "powershell"):
			return "powershell"
		}
	}
	return "generic"
}
