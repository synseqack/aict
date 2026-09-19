package mcpserver

// outputFlags are the output-shaping flags every aict tool accepts. They are
// not part of any tool's Config struct and so never appear in a generated
// input schema; declaring them once here keeps the per-tool tables focused on
// behaviour. The MCP server forces --json, so --xml and --plain are excluded
// deliberately.
var outputFlags = map[string]string{
	"nocompact": "--no-compact",
	"pretty":    "--pretty",
	"dict":      "--dict",
}

// flagMappings translates a tool's input-schema property to the CLI flag that
// aict's own parseFlags recognises. Property names are the lowercased Config
// field names that internal/tool.GenerateSchema advertises, so a mismatch
// here is a dead mapping that TestFlagMappingsMatchSchema catches.
//
// Only flags that actually change output are listed. A flag that a tool
// accepts but discards is omitted rather than advertised; the Config structs
// in tools/ drop the flag tag for the same reason. Both keep the input
// schema from promising behaviour aict does not have.
var flagMappings = map[string]map[string]string{
	"awk": {
		"fieldsep": "-F",
		"program":  "-f",
	},
	"checksums": {
		"algorithms": "-a",
	},
	// The checksums package registers these as separate commands that each
	// default to one algorithm; -a still selects which are reported.
	"md5sum":    {"algorithms": "-a"},
	"sha1sum":   {"algorithms": "-a"},
	"sha256sum": {"algorithms": "-a"},
	"cut": {
		"fields":     "-f",
		"delimiter":  "-d",
		"characters": "-c",
		"onlydelim":  "-s",
	},
	"diff": {
		"unified":        "-u",
		"recursive":      "-r",
		"ignoreallspace": "-w",
		"quiet":          "-q",
		"labelold":       "--label",
		"labelnew":       "--label",
	},
	"du": {
		"summarize": "-s",
		"all":       "-a",
		"maxdepth":  "--max-depth",
	},
	"file": {
		"brief": "-b",
		"mime":  "-i",
	},
	"find": {
		"name":     "-name",
		"type":     "-type",
		"mtime":    "-mtime",
		"size":     "-size",
		"maxdepth": "-maxdepth",
		"invert":   "-not",
		"or":       "-o",
	},
	"grep": {
		"recursive":        "-r",
		"linenumbers":      "-n",
		"fileswithmatches": "-l",
		"caseinsensitive":  "-i",
		"wordmatch":        "-w",
		"aftercontext":     "-A",
		"beforecontext":    "-B",
		"contextlines":     "-C",
		"countonly":        "-c",
		"invertmatch":      "-v",
		"fixedstrings":     "-F",
		"include":          "--include",
		"excludedir":       "--exclude-dir",
		"maxcount":         "-m",
		"workers":          "--workers",
	},
	"head": {
		"lines": "-n",
		"bytes": "-c",
	},
	"jq": {
		"path": "-p",
		"raw":  "-r",
	},
	"ls": {
		"all":       "-a",
		"almostall": "-A",
		"sorttime":  "-t",
		"reverse":   "-r",
		"recursive": "-R",
	},
	"sed": {
		"suppress":      "-n",
		"script":        "-e",
		"extendedregex": "-E",
	},
	"sort": {
		"numeric":   "-n",
		"reverse":   "-r",
		"key":       "-k",
		"delimiter": "-t",
		"unique":    "-u",
	},
	"stat": {
		"followsymlinks": "-L",
	},
	"tail": {
		"lines": "-n",
		"bytes": "-c",
	},
	"tar": {
		"extract": "-x",
	},
	"tr": {
		"delete":  "-d",
		"squeeze": "-s",
	},
	"uniq": {
		"count":      "-c",
		"duplicates": "-d",
		"unique":     "-u",
		"ignorecase": "-i",
	},
	"wc": {
		"lines":    "-l",
		"words":    "-w",
		"bytes":    "-c",
		"maxlines": "-L",
	},
	// cat, completions, df, env, ps, pwd, basename, dirname,
	// realpath, git, doctor, system take only output flags and positionals.
}

// positional describes one input that reaches the tool as a positional CLI
// argument rather than a flag.
type positional struct {
	property string // schema property name
	desc     string
	array    bool // true: a JSON array expands to repeated positionals
}

// positionalInputs declares, per tool, the properties that become positional
// arguments and the order the tool's argv requires.
//
// This table exists because a JSON object carries no order. Without it the
// emitted argv was nondeterministic, which reversed `diff a b` and shuffled
// `cat f1 f2` on roughly one call in five. Tools whose only inputs are flags
// or stdin are absent.
var positionalInputs = map[string][]positional{
	"awk":         {{"files", "Input files (default: stdin)", true}},
	"basename":    {{"names", "Paths to reduce", true}, {"suffix", "Suffix to strip from each name", false}},
	"cat":         {{"files", "Files to read", true}},
	"checksums":   {{"files", "Files to hash", true}},
	"completions": {{"shell", "Shell: bash, zsh, or fish", false}},
	"cut":         {{"files", "Input files (default: stdin)", true}},
	"diff":        {{"old", "Original file or directory", false}, {"new", "Changed file or directory", false}},
	"du":          {{"paths", "Paths to measure (default: .)", true}},
	"file":        {{"files", "Files to classify", true}},
	"find":        {{"root", "Directory to search (default: .)", false}},
	"grep":        {{"pattern", "Search pattern (regex or literal)", false}, {"root", "Directory to search (default: .)", false}},
	"head":        {{"files", "Files to read", true}},
	"jq":          {{"files", "JSON files to read (default: stdin)", true}},
	"ls":          {{"paths", "Paths to list (default: .)", true}},
	"realpath":    {{"paths", "Paths to resolve", true}},
	"sed":         {{"files", "Input files (default: stdin)", true}},
	"sort":        {{"files", "Input files (default: stdin)", true}},
	"stat":        {{"paths", "Paths to inspect (default: .)", true}},
	"tail":        {{"files", "Files to read", true}},
	"tar":         {{"archive", "Archive to read", false}},
	"tr":          {{"set1", "Characters to translate from", false}, {"set2", "Characters to translate to", false}},
	"uniq":        {{"files", "Input files (default: stdin)", true}},
	"wc":          {{"paths", "Files to count (default: .)", true}},
	"dirname":     {{"paths", "Paths to reduce", true}},
	"git":         {{"subcommand", "status, diff, log, ls-files, blame, or show", false}, {"args", "Arguments forwarded to git", true}},
}

// positionalOrder returns the declared positionals for a tool, or nil.
func positionalOrder(toolName string) []positional {
	return positionalInputs[toolName]
}

// positionalProperty reports whether a schema property is a declared
// positional for the tool, and if so whether it expands from an array.
func positionalProperty(toolName, property string) (array bool, ok bool) {
	for _, p := range positionalInputs[toolName] {
		if p.property == property {
			return p.array, true
		}
	}
	return false, false
}

// flagFor resolves a property to its CLI flag, checking the tool's own table
// before the output flags shared by every tool.
func flagFor(toolName, lowerKey string) (string, bool) {
	if flag, ok := flagMappings[toolName][lowerKey]; ok {
		return flag, true
	}
	if flag, ok := outputFlags[lowerKey]; ok {
		return flag, true
	}
	return "", false
}

// mergePositionalSchema advertises the positional inputs a tool accepts, so
// an agent can discover how to pass a path or a pattern. GenerateSchema only
// sees Config struct fields, and positionals never live there — without this
// the properties exist in no tool's schema, so callers had to guess them.
func mergePositionalSchema(toolName string, schema map[string]interface{}) {
	positionals := positionalOrder(toolName)
	if len(positionals) == 0 {
		return
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		props = make(map[string]interface{})
		schema["properties"] = props
	}
	for _, p := range positionals {
		if _, exists := props[p.property]; exists {
			continue
		}
		props[p.property] = map[string]interface{}{
			"type":        "string",
			"description": p.desc,
		}
		if p.array {
			props[p.property] = map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": p.desc,
			}
		}
	}
}
