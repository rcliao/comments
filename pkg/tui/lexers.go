package tui

// In-repo chroma lexers for the fence languages our docs actually carry:
// DBML — the data-model notation the design templates mandate — and a
// deliberately minimal mermaid (keywords,
// arrows, node ids; a readable fence, not a grammar).

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

func init() {
	lexers.Register(chroma.MustNewLexer(
		&chroma.Config{
			Name:      "DBML",
			Aliases:   []string{"dbml"},
			Filenames: []string{"*.dbml"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {
					{Pattern: `//.*`, Type: chroma.Comment, Mutator: nil},
					{Pattern: `\b(Table|Ref|Enum|Project|Note|TableGroup)\b`, Type: chroma.Keyword, Mutator: nil},
					{Pattern: `\[(pk|primary key|unique|not null|null|increment|default:[^\]]*|note:[^\]]*|ref:[^\]]*)\]`, Type: chroma.NameAttribute, Mutator: nil},
					{Pattern: `\b(int|integer|string|text|varchar|bool|boolean|float|double|decimal|timestamp|datetime|date|json|uuid|bigint|serial)\b`, Type: chroma.KeywordType, Mutator: nil},
					{Pattern: `[{}\[\]():,]`, Type: chroma.Punctuation, Mutator: nil},
					{Pattern: `[<>-]+`, Type: chroma.Operator, Mutator: nil},
					{Pattern: `"[^"]*"|'[^']*'`, Type: chroma.LiteralString, Mutator: nil},
					{Pattern: `\s+`, Type: chroma.Text, Mutator: nil},
					{Pattern: `[^\s{}\[\]():,<>-]+`, Type: chroma.Name, Mutator: nil},
				},
			}
		},
	))

	lexers.Register(chroma.MustNewLexer(
		&chroma.Config{
			Name:    "Mermaid",
			Aliases: []string{"mermaid"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {
					{Pattern: `%%.*`, Type: chroma.Comment, Mutator: nil},
					{Pattern: `\b(graph|flowchart|sequenceDiagram|erDiagram|classDiagram|stateDiagram|participant|subgraph|end|direction|LR|TD|TB|RL|BT|as)\b`, Type: chroma.Keyword, Mutator: nil},
					{Pattern: `-->|---|-\.->|==>|->>|-->>|\|\|--|\}o--|o\{|\.\.>`, Type: chroma.Operator, Mutator: nil},
					{Pattern: `\|[^|]*\||"[^"]*"`, Type: chroma.LiteralString, Mutator: nil},
					{Pattern: `[\[\](){}:;]`, Type: chroma.Punctuation, Mutator: nil},
					{Pattern: `\s+`, Type: chroma.Text, Mutator: nil},
					{Pattern: `[^\s\[\](){}:;|"-]+`, Type: chroma.Name, Mutator: nil},
				},
			}
		},
	))
}
