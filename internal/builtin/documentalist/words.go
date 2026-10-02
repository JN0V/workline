package documentalist

import (
	"regexp"
	"strings"
)

// Words as the removal rule compares them, in the doc's language. Its
// glue and fact words were English, and its words split on typography: on
// les-emplois' French templates, 26 one-line rewordings were refused for an
// apostrophe made typographic alone, 14 for an article mended
// (docs/research/documentalist-genericity.md).

// language is the project's `language` setting: "" reads each doc's own.
// Set once a run, from the settings (useLanguage).
var language string

// useLanguage takes the project's `language`.
func useLanguage(s Settings) { language = strings.ToLower(s.Language) }

// plainer makes typography plain: apostrophes, quotes, and the spaces a
// typographer writes (no-break, narrow, thin, figure) as a keyboard does.
var plainer = strings.NewReplacer("’", "'", "‘", "'", "ʼ", "'", "´", "'",
	"“", "\"", "”", "\"", "«", "\"", "»", "\"", "„", "\"",
	"\u00a0", " ", "\u202f", " ", "\u2009", " ", "\u2007", " ")

// digitGroup is a thousands' space inside a number: 1 000, 12 345 678.
var digitGroup = regexp.MustCompile(`(\d) (\d{3})\b`)

// plain is a line as its words are compared: typography plain, a number
// written whole ("1 000" as "1000").
func plain(s string) string {
	s = plainer.Replace(s)
	for {
		t := digitGroup.ReplaceAllString(s, "${1}${2}")
		if t == s {
			return s
		}
		s = t
	}
}

// elision is a French word elided before another: l'usager, d'accès,
// qu'il; the article, and the word, are two.
var elision = regexp.MustCompile(`(?i)^(l|d|j|m|n|s|t|c|qu|jusqu|lorsqu|puisqu)'(.+)$`)

// wordsOf are a line's words, typography plain, an elided French article
// apart from its word.
func wordsOf(line, lang string) []string {
	ws := wordOf.FindAllString(plain(line), -1)
	if lang != "fr" {
		return ws
	}
	var out []string
	for _, w := range ws {
		if m := elision.FindStringSubmatch(w); m != nil {
			out = append(out, m[1]+"'", m[2])
			continue
		}
		out = append(out, w)
	}
	return out
}

// marks are the words most frequent in each language, by which a doc's own
// is told.
var marks = map[string]map[string]bool{
	"en": wordSet(`the and is are of to with for this that it be on not`),
	"fr": wordSet(`le la les des est et une un pour dans avec du que qui sont pas ne sur au`),
}

// docLanguage is the language whose glue and fact words a doc's lines are
// read with: the project's, else the one whose most frequent words the doc
// says more; English when neither.
func docLanguage(content string) string {
	if glue[language] != nil {
		return language
	}
	n := map[string]int{}
	for _, w := range wordOf.FindAllString(strings.ToLower(plain(content)), -1) {
		for lang, m := range marks {
			if m[w] {
				n[lang]++
			}
		}
	}
	if n["fr"] > n["en"] {
		return "fr"
	}
	return "en"
}
