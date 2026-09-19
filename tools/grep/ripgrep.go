package grep

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/synseqack/aict/internal/detect"
	"github.com/synseqack/aict/internal/meta"
	"github.com/synseqack/aict/internal/ripgrep"
	xmlout "github.com/synseqack/aict/internal/xml"
)

// useRipgrep reports whether the request can be handed to an external
// ripgrep binary. Every flag aict supports translates to ripgrep except the
// context flags: ripgrep reports surrounding lines as separate messages that
// cannot be reassembled into aict's Before/After fields without diverging
// from the built-in engine on adjacent matches, so those searches stay
// in-process.
func useRipgrep(cfg Config) bool {
	if !ripgrep.Available() {
		return false
	}
	return cfg.BeforeContext == 0 && cfg.AfterContext == 0 && cfg.ContextLines == 0
}

// searchWithRipgrep runs the search through ripgrep. The second return is
// false when the caller should fall back to the built-in engine: ripgrep was
// unavailable, or the invocation failed outright.
func searchWithRipgrep(absPath, givenPath string, info os.FileInfo, cfg Config) (*GrepResult, bool) {
	result := &GrepResult{
		Pattern:       cfg.Pattern,
		Recursive:     xmlout.Bool(cfg.Recursive),
		CaseSensitive: xmlout.Bool(!cfg.CaseInsensitive),
		SearchRoot:    givenPath,
		Timestamp:     meta.Now(),
	}

	if cfg.Recursive && info.IsDir() {
		candidates, searched := candidatePaths(absPath, cfg)
		result.SearchedFiles = searched

		// ripgrep contributes match content only. aict's own file discovery
		// decides what gets searched, so include, exclude-dir, binary and
		// hidden-file handling stays identical to the built-in engine.
		searchable := make([]string, 0, len(candidates))
		for _, p := range candidates {
			if _, isBinary, _ := detect.DetectFromFile(p); !isBinary {
				searchable = append(searchable, p)
			}
		}
		if len(searchable) == 0 {
			return result, true
		}

		res, err := ripgrep.Search(context.Background(), rgArgs(cfg), searchable)
		if err != nil {
			return nil, false
		}

		fillFromRipgrep(result, res, absPath, cfg, searchable)
		sort.Slice(result.Matches, func(i, j int) bool {
			return result.Matches[i].Path < result.Matches[j].Path
		})
		return result, true
	}

	// Single file, or a directory without -r: the built-in engine reports
	// one searched file and no matches for a directory.
	result.SearchedFiles = 1
	if info.IsDir() {
		return result, true
	}

	if _, isBinary, _ := detect.DetectFromFile(absPath); isBinary {
		return result, true
	}
	if cfg.Include != "" {
		if matched, _ := filepath.Match(cfg.Include, filepath.Base(absPath)); !matched {
			return result, true
		}
	}

	res, err := ripgrep.Search(context.Background(), rgArgs(cfg), []string{absPath})
	if err != nil {
		return nil, false
	}

	var matches []GrepMatch
	if len(res.Files) > 0 {
		for _, m := range res.Files[0].Matches {
			matches = append(matches, GrepMatch{
				Number:      m.LineNumber,
				Text:        m.Text,
				OffsetBytes: m.AbsoluteOffset,
			})
		}
	}
	if len(matches) > 0 {
		result.MatchedFiles = 1
		result.TotalMatches = len(matches)
		result.Matches = append(result.Matches, GrepFileMatch{
			Path:          givenPath,
			Absolute:      absPath,
			Language:      detect.LanguageFromFile(absPath),
			MatchesInFile: len(matches),
			Lines:         matches,
		})
	}
	return result, true
}

func rgArgs(cfg Config) ripgrep.Args {
	return ripgrep.Args{
		Pattern:         cfg.Pattern,
		CaseInsensitive: cfg.CaseInsensitive,
		WordMatch:       cfg.WordMatch,
		FixedStrings:    cfg.FixedStrings,
		InvertMatch:     cfg.InvertMatch,
		MaxCount:        cfg.MaxCount,
	}
}

// fillFromRipgrep converts decoded matches into GrepFileMatch entries. With
// --files-with-matches aict lists every searched file, carrying its matches
// when it has any; otherwise only files with matches appear.
func fillFromRipgrep(result *GrepResult, res *ripgrep.Result, dirPath string, cfg Config, searchable []string) {
	byPath := make(map[string][]ripgrep.Match, len(res.Files))
	for _, f := range res.Files {
		byPath[f.Path] = f.Matches
	}

	for _, abs := range searchable {
		matches, found := byPath[abs]
		if !found && !cfg.FilesWithMatches {
			continue
		}

		// Stay nil when there is nothing to report: the built-in engine's
		// slice is nil for a matchless file, and nil vs [] is visible in JSON.
		var lines []GrepMatch
		for _, m := range matches {
			lines = append(lines, GrepMatch{
				Number:      m.LineNumber,
				Text:        m.Text,
				OffsetBytes: m.AbsoluteOffset,
			})
		}

		relPath, _ := filepath.Rel(dirPath, abs)

		result.Matches = append(result.Matches, GrepFileMatch{
			Path:          relPath,
			Absolute:      abs,
			Language:      detect.LanguageFromFile(abs),
			MatchesInFile: len(lines),
			Lines:         lines,
		})
		result.MatchedFiles++
		result.TotalMatches += len(lines)
	}
}

// candidatePaths walks dirPath applying aict's own include and exclude-dir
// rules, returning every file the built-in engine would have considered.
// The count it returns is searched_files: the built-in engine counts a file
// once it passes the include filter, before the binary check. Only regular
// files are returned; a device or FIFO would otherwise stall detection.
func candidatePaths(dirPath string, cfg Config) (paths []string, searched int) {
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if cfg.ExcludeDir != "" {
				if matched, _ := filepath.Match(cfg.ExcludeDir, info.Name()); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if cfg.Include != "" {
			if matched, _ := filepath.Match(cfg.Include, info.Name()); !matched {
				return nil
			}
		}
		searched++
		paths = append(paths, path)
		return nil
	})
	return paths, searched
}
