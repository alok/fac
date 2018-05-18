package conflict

import (
	"bytes"
	"sort"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/mkchoi212/fac/color"
)

// Conflict represents a single conflict that may have occurred
type Conflict struct {
	File *File

	Choice int
	Start  int
	Middle int
	End    int
	Diff3  []int

	LocalLines           []string
	LocalPureLines       []string
	IncomingLines        []string
	ColoredLocalLines    []string
	ColoredIncomingLines []string

	CurrentName string
	ForeignName string

	TopPeek    int
	BottomPeek int
}

// Supported git conflict styles
const (
	text = iota
	start
	diff3
	separator
	end
)

const (
	Local    = 1
	Incoming = 2
)

// IdentifyStyle identifies the conflict marker style of provided text
func IdentifyStyle(line string) (style int) {
	line = strings.TrimSpace(line)

	if strings.Contains(line, "<<<<<<<") {
		style = start
	} else if strings.Contains(line, ">>>>>>>") {
		style = end
	} else if line == "||||||| merged common ancestors" {
		style = diff3
	} else if line == "=======" {
		style = separator
	} else {
		style = text
	}
	return
}

// Valid checks if the parsed conflict has corresponding begin, separator,
// and middle line numbers
func (c *Conflict) Valid() bool {
	return c.File != nil && c.Middle != 0 && c.End != 0
}

// Equal checks if two `Conflict`s are equal
func (c Conflict) Equal(c2 *Conflict) bool {
	return c.File.AbsolutePath == c2.File.AbsolutePath && c.Start == c2.Start
}

// Extract extracts lines where conflicts exist
// and corresponding branch names
func (c *Conflict) Extract(lines []string) error {
	c.LocalLines = lines[c.Start+1 : c.Middle]
	if len(c.Diff3) != 0 {
		sort.Ints(c.Diff3)
		diff3Barrier := c.Diff3[0]
		c.LocalPureLines = lines[c.Start+1 : diff3Barrier]
	} else {
		c.LocalPureLines = c.LocalLines
	}
	c.IncomingLines = lines[c.Middle+1 : c.End]
	c.CurrentName = strings.Split(lines[c.Start], " ")[1]
	c.ForeignName = strings.Split(lines[c.End], " ")[1]
	return nil
}

// PaddingLines returns top and bottom padding lines based on
// `TopPeek` and `BottomPeek` values
func (c *Conflict) PaddingLines() (topPadding, bottomPadding []string) {
	lines := c.File.Lines
	start, end := c.Start, c.End

	if c.TopPeek >= start {
		c.TopPeek = start
	} else if c.TopPeek < 0 {
		c.TopPeek = 0
	}

	for _, l := range lines[start-c.TopPeek : start] {
		topPadding = append(topPadding, color.Black(color.Regular, l))
	}

	if c.BottomPeek >= len(lines)-c.End {
		c.BottomPeek = len(lines) - c.End
	} else if c.BottomPeek < 0 {
		c.BottomPeek = 0
	}

	for _, l := range lines[end : end+c.BottomPeek] {
		bottomPadding = append(bottomPadding, color.Black(color.Regular, l))
	}
	return
}

// HighlightSyntax highlights the stored file lines; both local and incoming lines
// The highlighted versions of the lines are stored in Conflict.Colored____Lines
// If the file extension is not supported, no highlights are applied
func (c *Conflict) HighlightSyntax() error {
	var lexer chroma.Lexer

	if lexer = lexers.Match(c.File.Name); lexer == nil {
		for _, block := range [][]string{c.LocalLines, c.IncomingLines} {
			if trial := lexers.Analyse(strings.Join(block, "")); trial != nil {
				lexer = trial
				break
			}
		}
	}

	if lexer == nil {
		lexer = lexers.Fallback
		c.ColoredLocalLines = c.LocalLines
		c.ColoredIncomingLines = c.IncomingLines
		return nil
	}

	style := styles.Get("emacs")
	formatter := formatters.Get("terminal")

	var it chroma.Iterator
	var err error
	buf := new(bytes.Buffer)
	var colorLine string

tokenizer:
	for i, block := range [][]string{c.LocalLines, c.IncomingLines} {
		for _, line := range block {
			if IdentifyStyle(line) == diff3 {
				colorLine = color.Red(color.Regular, line)
			} else {
				if it, err = lexer.Tokenise(nil, line); err != nil {
					break tokenizer
				}
				if err = formatter.Format(buf, style, it); err != nil {
					break tokenizer
				}
				colorLine = buf.String()
			}

			if i == 0 {
				c.ColoredLocalLines = append(c.ColoredLocalLines, colorLine)
			} else {
				c.ColoredIncomingLines = append(c.ColoredIncomingLines, colorLine)
			}
			buf.Reset()
		}
	}
	return err
}
