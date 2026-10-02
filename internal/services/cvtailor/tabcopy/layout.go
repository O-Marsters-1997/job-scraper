package tabcopy

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	defaultFont    = "Arial"
	defaultSize    = 11.0
	defaultSpacing = 100.0
	defaultPageW   = 612.0
	defaultPageH   = 792.0
	defaultMargin  = 72.0
	boldWeight     = 700
	defaultBullet  = "•"
	paragraphEnd   = "\n"
)

// Block is a laid-out paragraph and the Doc body indices it spans.
type Block struct {
	dto.LayoutBlock
	StartIndex, EndIndex int
}

// Document is a Tab's page geometry and paragraphs, resolved to pt.
type Document struct {
	Page   dto.LayoutPage
	Blocks []Block
}

type dimension struct {
	Magnitude float64 `json:"magnitude"`
}

type rgbColor struct {
	Color struct {
		RGB struct {
			Red, Green, Blue float64
		} `json:"rgbColor"`
	} `json:"color"`
}

func (c rgbColor) hex() string {
	rgb := c.Color.RGB
	if rgb.Red == 0 && rgb.Green == 0 && rgb.Blue == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", channel(rgb.Red), channel(rgb.Green), channel(rgb.Blue))
}

func channel(v float64) int { return int(math.Round(v * 255)) }

type borderStyle struct {
	Color     rgbColor  `json:"color"`
	Width     dimension `json:"width"`
	Padding   dimension `json:"padding"`
	DashStyle string    `json:"dashStyle"`
}

func (b *borderStyle) layout() *dto.LayoutBorder {
	if b == nil || b.Width.Magnitude == 0 {
		return nil
	}
	dash := "solid"
	switch b.DashStyle {
	case "DOT":
		dash = "dotted"
	case "DASH":
		dash = "dashed"
	}
	color := b.Color.hex()
	if color == "" {
		color = "#000000"
	}
	return &dto.LayoutBorder{Width: b.Width.Magnitude, Color: color, Padding: b.Padding.Magnitude, Dash: dash}
}

type paragraphLayout struct {
	Alignment       string       `json:"alignment"`
	LineSpacing     float64      `json:"lineSpacing"`
	SpaceAbove      dimension    `json:"spaceAbove"`
	SpaceBelow      dimension    `json:"spaceBelow"`
	IndentStart     dimension    `json:"indentStart"`
	IndentFirstLine dimension    `json:"indentFirstLine"`
	BorderTop       *borderStyle `json:"borderTop"`
	BorderBottom    *borderStyle `json:"borderBottom"`
	TabStops        []struct {
		Offset    dimension `json:"offset"`
		Alignment string    `json:"alignment"`
	} `json:"tabStops"`
}

type textLayout struct {
	FontSize           dimension            `json:"fontSize"`
	Bold               bool                 `json:"bold"`
	Italic             bool                 `json:"italic"`
	Underline          bool                 `json:"underline"`
	ForegroundColor    rgbColor             `json:"foregroundColor"`
	Link               struct{ URL string } `json:"link"`
	WeightedFontFamily struct {
		FontFamily string `json:"fontFamily"`
		Weight     int    `json:"weight"`
	} `json:"weightedFontFamily"`
}

// Layout resolves src's single-column body into pt-accurate blocks. It
// returns ErrUnsupported for the content Requests cannot rebuild.
func Layout(src json.RawMessage) (Document, error) {
	dt, named, err := decode(src)
	if err != nil {
		return Document{}, err
	}
	out := Document{Page: page(dt.DocumentStyle)}
	counters := map[string]int{}
	for _, el := range dt.Body.Content {
		p, err := paragraphOf(el)
		if err != nil {
			return Document{}, err
		}
		if p == nil {
			continue
		}
		block := blockOf(p, named, dt.Lists)
		if p.Bullet != nil {
			key := p.Bullet.ListID + "/" + strconv.Itoa(p.Bullet.NestingLevel)
			counters[key]++
			block.Bullet = bulletOf(p, dt.Lists, counters[key], block.Runs)
		}
		out.Blocks = append(out.Blocks, Block{LayoutBlock: block, StartIndex: el.StartIndex, EndIndex: el.EndIndex})
	}
	return out, nil
}

func blockOf(p *paragraph, named map[string]namedStyle, lists map[string]list) dto.LayoutBlock {
	pl := decodeStyle[paragraphLayout](paragraphStyleOf(p, named, lists))
	block := dto.LayoutBlock{
		Align:           alignOf(pl.Alignment),
		LineSpacing:     cmp.Or(pl.LineSpacing, defaultSpacing),
		SpaceAbove:      pl.SpaceAbove.Magnitude,
		SpaceBelow:      pl.SpaceBelow.Magnitude,
		IndentStart:     pl.IndentStart.Magnitude,
		IndentFirstLine: pl.IndentFirstLine.Magnitude,
		BorderTop:       pl.BorderTop.layout(),
		BorderBottom:    pl.BorderBottom.layout(),
		Runs:            runsOf(p, named),
	}
	for _, ts := range pl.TabStops {
		block.TabStops = append(block.TabStops, dto.LayoutTab{Offset: ts.Offset.Magnitude, Alignment: tabAlignment(ts.Alignment)})
	}
	return block
}

func bulletOf(p *paragraph, lists map[string]list, n int, runs []dto.LayoutRun) *dto.LayoutBullet {
	lvl := levelOf(lists, p.Bullet.ListID, p.Bullet.NestingLevel)
	b := &dto.LayoutBullet{Glyph: lvl.glyph(n), Level: p.Bullet.NestingLevel}
	if len(runs) > 0 {
		b.Size = runs[0].Size
	}
	return b
}

func runsOf(p *paragraph, named map[string]namedStyle) []dto.LayoutRun {
	namedType := p.ParagraphStyle.namedType()
	var runs []dto.LayoutRun
	for _, pe := range p.Elements {
		ts := decodeStyle[textLayout](textStyleOf(named, namedType, pe.TextRun.TextStyle))
		runs = append(runs, dto.LayoutRun{
			Text:      pe.TextRun.Content,
			Font:      cmp.Or(ts.WeightedFontFamily.FontFamily, defaultFont),
			Size:      cmp.Or(ts.FontSize.Magnitude, defaultSize),
			Bold:      ts.Bold || ts.WeightedFontFamily.Weight >= boldWeight,
			Italic:    ts.Italic,
			Underline: ts.Underline,
			Color:     ts.ForegroundColor.hex(),
			Link:      ts.Link.URL,
		})
	}
	if len(runs) == 0 {
		return runs
	}
	last := len(runs) - 1
	runs[last].Text = strings.TrimSuffix(runs[last].Text, paragraphEnd)
	switch {
	case runs[last].Text != "":
	case last == 0:
		runs[0].Text = paragraphEnd
	default:
		runs = runs[:last]
	}
	return runs
}

func (l nestingLevel) glyph(n int) string {
	if l.GlyphType == glyphDecimal {
		return strconv.Itoa(n) + "."
	}
	if l.GlyphSymbol != "" {
		return l.GlyphSymbol
	}
	return defaultBullet
}

func alignOf(a string) string {
	switch a {
	case "CENTER":
		return "center"
	case "END":
		return "right"
	case "JUSTIFIED":
		return "justify"
	}
	return "left"
}

func tabAlignment(a string) string {
	switch a {
	case "CENTER":
		return "center"
	case "END":
		return "end"
	}
	return "start"
}

func page(ds docStyle) dto.LayoutPage {
	d := decodeStyle[struct {
		PageSize     struct{ Width, Height dimension }
		MarginTop    dimension
		MarginBottom dimension
		MarginLeft   dimension
		MarginRight  dimension
	}](ds)
	return dto.LayoutPage{
		Width:        cmp.Or(d.PageSize.Width.Magnitude, defaultPageW),
		Height:       cmp.Or(d.PageSize.Height.Magnitude, defaultPageH),
		MarginTop:    cmp.Or(d.MarginTop.Magnitude, defaultMargin),
		MarginBottom: cmp.Or(d.MarginBottom.Magnitude, defaultMargin),
		MarginLeft:   cmp.Or(d.MarginLeft.Magnitude, defaultMargin),
		MarginRight:  cmp.Or(d.MarginRight.Magnitude, defaultMargin),
	}
}

func decodeStyle[T any](s any) T {
	var v T
	b, err := json.Marshal(s)
	if err == nil {
		_ = json.Unmarshal(b, &v)
	}
	return v
}
