package detect

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

var extMap = map[string]string{
	".go":           "go",
	".mod":          "go",
	".sum":          "",
	".py":           "python",
	".pyw":          "python",
	".pyi":          "python",
	".pyx":          "python",
	".rs":           "rust",
	".ts":           "typescript",
	".tsx":          "typescript",
	".mts":          "typescript",
	".cts":          "typescript",
	".js":           "javascript",
	".jsx":          "javascript",
	".mjs":          "javascript",
	".cjs":          "javascript",
	".java":         "java",
	".rb":           "ruby",
	".rbw":          "ruby",
	".rake":         "ruby",
	".gemspec":      "ruby",
	".c":            "c",
	".h":            "c",
	".cpp":          "cpp",
	".cc":           "cpp",
	".cxx":          "cpp",
	".c++":          "cpp",
	".hpp":          "cpp",
	".hxx":          "cpp",
	".h++":          "cpp",
	".cs":           "csharp",
	".fs":           "fsharp",
	".fsx":          "fsharp",
	".fsi":          "fsharp",
	".kt":           "kotlin",
	".kts":          "kotlin",
	".ktm":          "kotlin",
	".swift":        "swift",
	".scala":        "scala",
	".sc":           "scala",
	".sh":           "bash",
	".bash":         "bash",
	".zsh":          "zsh",
	".fish":         "fish",
	".yml":          "yaml",
	".yaml":         "yaml",
	".json":         "json",
	".jsonc":        "json",
	".jsonl":        "json",
	".json5":        "json",
	".toml":         "toml",
	".xml":          "xml",
	".html":         "html",
	".htm":          "html",
	".xhtml":        "html",
	".css":          "css",
	".scss":         "scss",
	".sass":         "sass",
	".less":         "less",
	".md":           "markdown",
	".mdx":          "markdown",
	".markdown":     "markdown",
	".txt":          "text",
	".sql":          "sql",
	".r":            "r",
	".R":            "r",
	".php":          "php",
	".php3":         "php",
	".php4":         "php",
	".php5":         "php",
	".phtml":        "php",
	".lua":          "lua",
	".pl":           "perl",
	".pm":           "perl",
	".t":            "perl",
	".pod":          "perl",
	".ex":           "elixir",
	".exs":          "elixir",
	".eex":          "elixir",
	".heex":         "elixir",
	".leex":         "elixir",
	".hs":           "haskell",
	".lhs":          "haskell",
	".cabal":        "haskell",
	".ml":           "ocaml",
	".mli":          "ocaml",
	".erl":          "erlang",
	".hrl":          "erlang",
	".escript":      "erlang",
	".clj":          "clojure",
	".cljc":         "clojure",
	".cljs":         "clojure",
	".edn":          "clojure",
	".proto":        "proto",
	".graphql":      "graphql",
	".gql":          "graphql",
	".tf":           "terraform",
	".tfvars":       "terraform",
	".hcl":          "hcl",
	".nomad":        "hcl",
	".nix":          "nix",
	".zig":          "zig",
	".zon":          "zig",
	".v":            "vlang",
	".dart":         "dart",
	".elm":          "elm",
	".vue":          "vue",
	".svelte":       "svelte",
	".ps1":          "powershell",
	".psm1":         "powershell",
	".psd1":         "powershell",
	".ps1xml":       "powershell",
	".bat":          "batch",
	".cmd":          "batch",
	".m":            "objc",
	".mm":           "objc",
	".vim":          "viml",
	".el":           "emacs-lisp",
	".jl":           "julia",
	".nim":          "nim",
	".nims":         "nim",
	".nimble":       "nim",
	".cr":           "crystal",
	".groovy":       "groovy",
	".gradle":       "groovy",
	".lock":         "",
	".d":            "d",
	".di":           "d",
	".purs":         "purescript",
	".sol":          "solidity",
	".move":         "move",
	".mojo":         "mojo",
	".🔥":            "mojo",
	".gleam":        "gleam",
	".roc":          "roc",
	".raku":         "raku",
	".pl6":          "raku",
	".pas":          "pascal",
	".pp":           "pascal",
	".lpr":          "pascal",
	".f":            "fortran",
	".f90":          "fortran",
	".f95":          "fortran",
	".f03":          "fortran",
	".for":          "fortran",
	".f77":          "fortran",
	".asm":          "assembly",
	".s":            "assembly",
	".S":            "assembly",
	".a51":          "assembly",
	".nasm":         "assembly",
	".cob":          "cobol",
	".cbl":          "cobol",
	".COB":          "cobol",
	".CBL":          "cobol",
	".lisp":         "lisp",
	".lsp":          "lisp",
	".scm":          "scheme",
	".ss":           "scheme",
	".rkt":          "racket",
	".tcl":          "tcl",
	".vbs":          "vbscript",
	".vba":          "vba",
	".cobol":        "cobol",
	".prg":          "xbase",
	".hx":           "haxe",
	".vala":         "vala",
	".vapi":         "vala",
	".wasm":         "webassembly",
	".wat":          "webassembly",
	".coffee":       "coffeescript",
	".iced":         "coffeescript",
	".pug":          "pug",
	".jade":         "pug",
	".styl":         "stylus",
	".handlebars":   "handlebars",
	".hbs":          "handlebars",
	".mustache":     "mustache",
	".twig":         "twig",
	".ejs":          "ejs",
	".erb":          "erb",
	".liquid":       "liquid",
	".csv":          "csv",
	".tsv":          "tsv",
	".log":          "text",
	".ini":          "ini",
	".cfg":          "text",
	".conf":         "text",
	".properties":   "properties",
	".env":          "text",
	".gitignore":    "text",
	".dockerignore": "text",
	".editorconfig": "text",
	".diff":         "diff",
	".patch":        "diff",
	".sh.in":        "bash",
}

var noExtBaseMap = map[string]string{
	"Dockerfile":     "dockerfile",
	"dockerfile":     "dockerfile",
	"Makefile":       "makefile",
	"makefile":       "makefile",
	"GNUmakefile":    "makefile",
	"Containerfile":  "dockerfile",
	"containerfile":  "dockerfile",
	"Rakefile":       "ruby",
	"rakefile":       "ruby",
	"Gemfile":        "ruby",
	"Vagrantfile":    "ruby",
	"Berksfile":      "ruby",
	"Podfile":        "ruby",
	"Jenkinsfile":    "groovy",
	"jenkinsfile":    "groovy",
	"CMakeLists.txt": "cmake",
	"LICENSE":        "text",
	"README":         "markdown",
	"CHANGELOG":      "markdown",
	"AUTHORS":        "text",
	"CONTRIBUTORS":   "text",
	"NOTICE":         "text",
	"Procfile":       "text",
	".gitconfig":     "text",
	".bashrc":        "bash",
	".bash_profile":  "bash",
	".zshrc":         "zsh",
	".zprofile":      "zsh",
	".zshenv":        "zsh",
	".profile":       "bash",
	".vimrc":         "viml",
	".gvimrc":        "viml",
	"_vimrc":         "viml",
	"_gvimrc":        "viml",
	".emacs":         "emacs-lisp",
	".xinitrc":       "bash",
	".xprofile":      "bash",
	".xsession":      "bash",
	".inputrc":       "text",
	".nanorc":        "text",
	".tmux.conf":     "text",
	".screenrc":      "text",
	".wgetrc":        "text",
	".curlrc":        "text",
	".gitattributes": "text",
	".gitmodules":    "text",
	".editorconfig":  "text",
}

var shebangMap = []struct {
	pattern string
	lang    string
}{
	{"python3", "python"},
	{"python2", "python"},
	{"python", "python"},
	{"node", "javascript"},
	{"nodejs", "javascript"},
	{"deno", "javascript"},
	{"bun", "javascript"},
	{"ruby", "ruby"},
	{"bash", "bash"},
	{"/sh", "bash"},
	{"zsh", "zsh"},
	{"fish", "fish"},
	{"perl", "perl"},
	{"php", "php"},
	{"rustc", "rust"},
	{"rustup", "rust"},
	{"go ", "go"},
	{"go\n", "go"},
	{"lua", "lua"},
	{"awk", "awk"},
	{"gawk", "awk"},
	{"sed", "sed"},
	{"tclsh", "tcl"},
	{"wish", "tcl"},
	{"Rscript", "r"},
	{"julia", "julia"},
	{"crystal", "crystal"},
	{"dart", "dart"},
	{"elixir", "elixir"},
	{"erl", "erlang"},
	{"escript", "erlang"},
	{"groovy", "groovy"},
	{"scala", "scala"},
	{"swift", "swift"},
	{"perl6", "raku"},
	{"raku", "raku"},
	{"rakudo", "raku"},
	{"powershell", "powershell"},
	{"pwsh", "powershell"},
	{"nu", "nu"},
}

func Language(path string) string {
	base := filepath.Base(path)

	if lang, ok := noExtBaseMap[base]; ok {
		return lang
	}

	ext := filepath.Ext(path)
	if ext != "" {
		if lang := LanguageFromExtension(ext); lang != "" {
			return lang
		}
	}

	if base != "" && ext == "" {
		if lang, ok := noExtBaseMap[base]; ok {
			return lang
		}
	}

	return ""
}

func LanguageFromExtension(ext string) string {
	if lang, ok := extMap[ext]; ok {
		return lang
	}

	lower := strings.ToLower(ext)
	if lang, ok := extMap[lower]; ok {
		return lang
	}

	return ""
}

func LanguageFromShebang(content []byte) string {
	if len(content) < 2 || content[0] != '#' || content[1] != '!' {
		return ""
	}

	idx := bytes.IndexByte(content, '\n')
	if idx == -1 {
		idx = len(content)
	}
	line := bytes.TrimSpace(content[:idx])

	if len(line) < 3 || line[0] != '#' || line[1] != '!' {
		return ""
	}

	line = line[2:]
	line = bytes.TrimSpace(line)

	if bytes.HasPrefix(line, []byte("/usr/bin/env ")) {
		line = bytes.TrimPrefix(line, []byte("/usr/bin/env "))
		line = bytes.TrimSpace(line)
		parts := bytes.Fields(line)
		if len(parts) > 0 {
			line = parts[0]
		}
	} else if bytes.HasPrefix(line, []byte("/usr/local/bin/env ")) {
		line = bytes.TrimPrefix(line, []byte("/usr/local/bin/env "))
		line = bytes.TrimSpace(line)
		parts := bytes.Fields(line)
		if len(parts) > 0 {
			line = parts[0]
		}
	}

	interpreter := strings.ToLower(string(line))

	for _, entry := range shebangMap {
		if interpreter == entry.pattern {
			return entry.lang
		}
	}

	for _, entry := range shebangMap {
		if strings.Contains(interpreter, entry.pattern) {
			return entry.lang
		}
	}

	return ""
}

func LanguageFromFile(path string) string {
	if lang := Language(path); lang != "" {
		return lang
	}

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return ""
	}

	return LanguageFromShebang(line)
}
