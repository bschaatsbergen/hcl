// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// This is a 'go generate'-oriented program for producing the "Variables"
// method on every Expression implementation found within this package.
// All expressions share the same implementation for this method, which
// just wraps the package-level function "Variables" and uses an AST walk
// to do its work.

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
)

func main() {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, ".", nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while parsing: %s\n", err)
		os.Exit(1)
	}
	pkg := pkgs["hclsyntax"]

	// Walk all the files and collect the receivers of any "Value" methods
	// that look like they are trying to implement Expression.
	var recvs []string
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fd.Name.Name != "Value" {
				continue
			}
			results := fd.Type.Results.List
			if len(results) != 2 {
				continue
			}
			valResult := fd.Type.Results.List[0].Type.(*ast.SelectorExpr).X.(*ast.Ident)
			diagsResult := fd.Type.Results.List[1].Type.(*ast.SelectorExpr).X.(*ast.Ident)

			if valResult.Name != "cty" && diagsResult.Name != "hcl" {
				continue
			}

			// If we have a method called Value and it returns something in
			// "cty" followed by something in "hcl" then that's specific enough
			// for now, even though this is not 100% exact as a correct
			// implementation of Value.

			recvTy := fd.Recv.List[0].Type

			switch rtt := recvTy.(type) {
			case *ast.StarExpr:
				name := rtt.X.(*ast.Ident).Name
				recvs = append(recvs, fmt.Sprintf("*%s", name))
			default:
				fmt.Fprintf(os.Stderr, "don't know what to do with a %T receiver\n", recvTy)
			}

		}
	}

	sort.Strings(recvs)

	of, err := os.OpenFile("expression_vars.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open output file: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprint(of, outputPreamble)
	for _, recv := range recvs {
		fmt.Fprintf(of, outputMethodFmt, recv)
	}
	fmt.Fprint(of, "\n")

}

const outputPreamble = `// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hclsyntax

// Generated by expression_vars_get.go. DO NOT EDIT.
// Run 'go generate' on this package to update the set of functions here.

import (
	"github.com/hashicorp/hcl/v2"
)`

const outputMethodFmt = `

func (e %s) Variables() []hcl.Traversal {
	return Variables(e)
}`
