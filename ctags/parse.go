package ctags

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

type ETag struct {
	File    string
	Def     string
	Name    string
	Line    int
	ByteOff int
}

const (
	sepTag = "\x7f"
	sepPos = "\x01"
	sepCol = ","
)

type ETagsParser struct {
	// input
	config *Config

	// output
	tags      []ETag
	langFiles map[string][]string

	// temporary state
	curFile string
}

func NewParser() (*ETagsParser, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}
	return &ETagsParser{
		langFiles: make(map[string][]string),
		config:    cfg,
	}, nil
}

func (p *ETagsParser) Units() []*unit.SourceUnit {
	units := make([]*unit.SourceUnit, 0, len(p.langFiles))
	for lang, files := range p.langFiles {
		u := &unit.SourceUnit{
			Key:  unit.Key{Version: "", Type: langUnitType(lang), Name: "."},
			Info: unit.Info{Files: files},
		}
		units = append(units, u)
	}
	return units
}

func langUnitType(lang string) string {
	// return fmt.Sprintf("%s-ctags", lang)
	return "PipPackage"
}

func (p *ETagsParser) Defs() []*Def {
	// keep track of which names that have already occurred in a file
	fileDefNames := make(map[string]map[string]int)
	for _, files := range p.langFiles {
		for _, file := range files {
			fileDefNames[file] = make(map[string]int)
		}
	}

	tags := p.Tags()
	defs := make([]*Def, 0, len(tags))
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		name := fmt.Sprintf("%s$%d", tag.Name, fileDefNames[tag.File][tag.Name])
		fileDefNames[tag.File][tag.Name]++

		nameIdx := strings.Index(tag.Def, tag.Name)
		if nameIdx < 0 {
			continue
		}
		defStart := tag.ByteOff + nameIdx
		defEnd := defStart + len(tag.Name)

		defs = append(defs, &Def{
			DefKey: graph.DefKey{
				UnitType: langUnitType(p.config.Lang(tag.File)),
				Unit:     ".",
				Path:     fmt.Sprintf("%s:%s", tag.File, name),
			},
			Name:     tag.Name,
			File:     tag.File,
			DefStart: uint32(defStart),
			DefEnd:   uint32(defEnd),
			Exported: true,
			Local:    false,
			Data:     defFormatDataFromTag(tag),
		})
	}
	return defs
}

func (p *ETagsParser) Refs() []*graph.Ref {
	defs := p.Defs()
	refs := make([]*graph.Ref, 0, len(defs))
	for _, def := range defs {
		refs = append(refs, &graph.Ref{
			DefRepo:     def.Repo,
			DefUnitType: def.UnitType,
			DefUnit:     def.Unit,
			DefPath:     def.Path,
			Repo:        def.Repo,
			CommitID:    def.CommitID,
			UnitType:    def.UnitType,
			Unit:        def.Unit,
			Def:         true,
			File:        def.File,
			Start:       def.DefStart,
			End:         def.DefEnd,
		})
	}
	return refs
}

func (p *ETagsParser) Tags() []ETag {
	return p.tags
}

func (p *ETagsParser) Parse(r *bufio.Reader) error {
	p.curFile = ""

	line, err := r.ReadString('\n')
	for ; err == nil; line, err = r.ReadString('\n') {
		if err := p.parseLine(strings.TrimRight(line, "\r\n")); err != nil {
			return err
		}
	}
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (p *ETagsParser) parseLine(line string) error {
	if len(strings.TrimSpace(line)) == 0 || strings.HasPrefix(line, "!") {
		return nil
	}

	nameIdx := strings.Index(line, sepTag)
	if nameIdx < 0 {
		// File line
		cmps := strings.Split(line, ",")
		if len(cmps) != 2 {
			return fmt.Errorf("tags line parsing error: unrecognized format, line was %q", line)
		}
		if _, err := strconv.Atoi(cmps[1]); err != nil {
			return fmt.Errorf("tags line parsing error: %s, line was %q", err, line)
		}

		file := cmps[0]
		lang := p.config.Lang(file)
		p.curFile = file
		p.langFiles[lang] = append(p.langFiles[lang], file)
		return nil
	}

	// Symbol line
	lineNoIdx_ := strings.Index(line[nameIdx:], sepPos)
	if lineNoIdx_ < 0 {
		return fmt.Errorf("tags line parsing error: could not find character %U, line was %q", sepPos, line)
	}
	lineNoIdx := nameIdx + lineNoIdx_

	colIdx_ := strings.Index(line[lineNoIdx:], sepCol)
	if colIdx_ < 0 {
		return fmt.Errorf("tags line parsing error: could not find character %q, line was %q", sepCol, line)
	}
	colIdx := lineNoIdx + colIdx_

	lineNo, err := strconv.Atoi(line[lineNoIdx+1 : colIdx])
	if err != nil {
		return fmt.Errorf("tags line parsing error: could not parse line number, line was %q", line)
	}
	colNo, err := strconv.Atoi(line[colIdx+1:])
	if err != nil {
		return fmt.Errorf("tags line parsing error: could not parse byte offset, line was %q", line)
	}

	p.tags = append(p.tags, ETag{
		File:    p.curFile,
		Def:     line[0:nameIdx],
		Name:    line[nameIdx+1 : lineNoIdx],
		Line:    lineNo,
		ByteOff: colNo,
	})
	return nil
}

// defFormatDataFromTag returns the display formatting data for a
// definition derived from the specified tag.
//
// Precondition: it assumes that tag.Name exists in tag.Def.
func defFormatDataFromTag(tag ETag) *DefFormatData {
	nameIdx := strings.Index(tag.Def, tag.Name)
	if nameIdx < 0 {
		log.Printf("! warn: name (%q) not found in definition %q", tag.Name, tag.Def)
		return nil
	}
	keyword := strings.TrimSpace(tag.Def[:nameIdx])
	typ := tag.Def[nameIdx+len(tag.Name):]
	sep := ""
	if len(typ) >= 1 && typ[0] == ':' {
		sep, typ = typ[0:1], strings.TrimSpace(typ[1:])
	}
	return &DefFormatData{
		Name:      tag.Name,
		Keyword:   keyword,
		Type:      typ,
		Kind:      keyword,
		Separator: sep,
	}
}

// This mirrors the format data (DefData) struct of Sourcegraph's
// basic def formatter. We don't depend directly on that because we
// should have no dependencies on Sourcegraph here.
type DefFormatData struct {
	Name      string
	Keyword   string
	Type      string
	Kind      string
	Separator string
}
