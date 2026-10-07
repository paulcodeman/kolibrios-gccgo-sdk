// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package go2go

import (
	"encoding/hex"
	"fmt"
	"go/types"
	"strings"
	"unicode"
)

// We use Oriya digit zero as a separator.
// Do not use this character in your own identifiers.
const nameSep = '୦'

// We use Oriya digit eight to introduce a special character code.
// Do not use this character in your own identifiers.
const nameIntro = '୮'

var nameCodes = map[rune]int{
	' ':       0,
	'*':       1,
	';':       2,
	',':       3,
	'{':       4,
	'}':       5,
	'[':       6,
	']':       7,
	'(':       8,
	')':       9,
	'.':       10,
	'<':       11,
	'-':       12,
	'/':       13,
	nameSep:   14,
	nameIntro: 15,
}

func compatMemberName(obj types.Object) string {
	return "GccgoCompatMember_" + hex.EncodeToString([]byte(obj.Pkg().Path())) + "_" + hex.EncodeToString([]byte(obj.Name()))
}

func compatImportName(path string) string {
	return "GccgoImport_" + hex.EncodeToString([]byte(path))
}

// instantiatedName returns the name of a newly instantiated function.
func (t *translator) instantiatedName(qid qualifiedIdent, args []types.Type) (string, error) {
	obj := t.findTypesObject(qid)
	if tn, ok := obj.(*types.TypeName); ok {
		pkg := tn.Pkg()
		display := func(p *types.Package) string { return p.Name() }
		identityArgs, displayArgs := make([]string, len(args)), make([]string, len(args))
		for i, arg := range args {
			arg = normalizeAliases(arg)
			identityArgs[i] = t.importer.typeIdentity(arg)
			displayArgs[i] = types.TypeString(arg, display)
		}
		identity := pkg.Path() + "." + tn.Name() + "[" + strings.Join(identityArgs, ",") + "]"
		name := tn.Name() + "[" + strings.Join(displayArgs, ",") + "]"
		// The native frontend decodes this reserved compiler name. Every
		// specialization carries its defining package and full Go identity.
		parts := []string{identity, pkg.Path(), pkg.Name(), name}
		for i, part := range parts {
			parts[i] = hex.EncodeToString([]byte(part))
		}
		return "GccgoCompatType_" + strings.Join(parts, "_"), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "instantiate%c", nameSep)
	if qid.pkg != nil {
		fmt.Fprint(&sb, hex.EncodeToString([]byte(qid.pkg.Path())))
	} else {
		fmt.Fprint(&sb, hex.EncodeToString([]byte(t.tpkg.Path())))
	}
	fmt.Fprintf(&sb, "%c%s", nameSep, qid.ident.Name)
	for _, typ := range args {
		sb.WriteRune(nameSep)
		s := t.importer.typeIdentity(normalizeAliases(typ))

		// We have to uniquely translate s into a valid Go identifier.
		// This is not possible in general but we assume that
		// identifiers will not contain nameSep or nameIntro.
		for _, r := range s {
			if (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') && r != nameSep && r != nameIntro {
				sb.WriteRune(r)
			} else {
				code, ok := nameCodes[r]
				if ok {
					fmt.Fprintf(&sb, "%c%x_", nameIntro, code)
				} else {
					fmt.Fprintf(&sb, "%cu%x_", nameIntro, r)
				}
			}
		}
	}
	return sb.String(), nil
}

// importableName returns a name that we define in each package, so that
// we have something to import to avoid an unused package error.
func (t *translator) importableName() string {
	return "Importable" + string(nameSep)
}
