package codepick

import (
	"slices"
	"testing"
)

// Each family's imports, by one row of data: the project's files found,
// a package from outside left out.
func TestImports(t *testing.T) {
	dir := repo(t, map[string]string{
		"go.mod":                 "module example.com/m\n",
		"app/app.go":             "package app\n\nimport (\n\t\"fmt\"\n\tm \"example.com/m/lib/money\"\n)\n",
		"lib/money/a.go":         "package money\n",
		"lib/money/b.go":         "package money\n",
		"lib/money/a_test.go":    "package money\n",
		"py/shop/cart.py":        "from .money import Amount\nimport shop.rates\nimport os\n",
		"py/shop/money.py":       "",
		"shop/rates.py":          "",
		"web/src/cart.ts":        "import { a } from './money.js';\nimport React from 'react';\nconst r = require('../lib/rates');\n",
		"web/src/money.ts":       "",
		"web/lib/rates/index.ts": "",
		"c/src/a.cpp":            "#include \"a.h\"\n#include <vector>\n#include <Lib/b.h>\n",
		"c/src/a.h":              "",
		"c/include/Lib/b.h":      "",
	})
	p, err := New(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for from, want := range map[string][]string{
		"app/app.go":      {"lib/money/a.go", "lib/money/b.go"},
		"py/shop/cart.py": {"py/shop/money.py", "shop/rates.py"},
		"web/src/cart.ts": {"web/src/money.ts", "web/lib/rates/index.ts"},
		"c/src/a.cpp":     {"c/src/a.h", "c/include/Lib/b.h"},
	} {
		var got []string
		for _, l := range p.imports(from, []string{"**/*_test.go"}) {
			got = append(got, l.file)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s imports %v, want %v", from, got, want)
		}
	}
}
