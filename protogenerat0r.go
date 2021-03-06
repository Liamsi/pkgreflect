/*
Problem: Go reflection does not support enumerating types, variables and functions of packages.

protogenerat0r generates a file named protogenerat0r.go in every parsed package directory.

Command line usage:

	protogenerat0r --help
	protogenerat0r [-notypes][-nofuncs][-novars][-unexported][-norecurs][-gofile=filename.go] [DIR_NAME]

protogenerat0r traverses recursively into sub-directories.
If no DIR_NAME is given, then the current directory is used as root.
*/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

var (
	stdout bool
	gofile string
)

func main() {

	flag.StringVar(&gofile, "gofile", "proto_generator.go", "Name of the generated .go file")
	flag.BoolVar(&stdout, "stdout", false, "Write to stdout.")
	flag.Parse()

	if len(flag.Args()) > 0 {
		for _, dir := range flag.Args() {
			parseDir(dir)
		}
	} else {
		parseDir(".")
	}
}

func parseDir(dir string) {
	dirFile, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	defer dirFile.Close()
	info, err := dirFile.Stat()
	if err != nil {
		panic(err)
	}
	if !info.IsDir() {
		panic("Path is not a directory: " + dir)
	}

	pkgs, err := parser.ParseDir(token.NewFileSet(), dir, filter, 0)
	if err != nil {
		panic(err)
	}

	for _, pkg := range pkgs {
		var buf bytes.Buffer

		_, _ = fmt.Fprintln(&buf, "// Code generated by github.com/Liamsi/protogenerat0r DO NOT EDIT.\n")
		_, _ = fmt.Fprintln(&buf, "package", pkg.Name)
		_, _ = fmt.Fprintln(&buf, "")
		_, _ = fmt.Fprint(&buf,
			`import (
	"os"

	// we use dedis' protobuf library to generate proto files from go-structs
	// see: https://github.com/dedis/protobuf#generating-proto-files
	"github.com/dedis/protobuf"
)

`)

		// Types
		fmt.Fprintln(&buf, "var structTypes = []interface{}{")
		printTypeLine(&buf, pkg, ast.Typ, "\t%s{},\n")
		fmt.Fprintln(&buf, "}")
		fmt.Fprintln(&buf, "")

		// enum map
		// TODO we probably do not need this!
		hasEnums := false
		if hasEnums {
			fmt.Fprintln(&buf, "var enumMap  = protobuf.EnumMap{")
			printEnumLine(&buf, pkg, ast.Typ, "\t\"%s\": *%s,\n")
			fmt.Fprintln(&buf, "}")
			fmt.Fprintln(&buf, "")
		}

		// Functions
		// TODO maybe this is useful for recognizing custom types?
		//
		//		fmt.Fprintln(&buf, "var Functions = map[string]reflect.Value{")
		//		printEnumLine(&buf, pkg, ast.Fun, "\t\"%s\": reflect.ValueOf(%s),\n")
		//		fmt.Fprintln(&buf, "}")
		//		fmt.Fprintln(&buf, "")
		//

		_, _ = fmt.Fprintln(&buf, `
// Call this method to generate protobuf messages: 
func GenerateProtos() {
	// see: https://github.com/dedis/protobuf#generating-proto-files
	protobuf.GenerateProtobufDefinition(os.Stdout, structTypes, nil, nil)
}`)

		if stdout {
			io.Copy(os.Stdout, &buf)
		} else {
			filename := filepath.Join(dir, gofile)
			newFileData := buf.Bytes()
			oldFileData, _ := ioutil.ReadFile(filename)
			if !bytes.Equal(newFileData, oldFileData) {
				err = ioutil.WriteFile(filename, newFileData, 0660)
				if err != nil {
					panic(err)
				}
			}
		}
	}

	//if !norecurs {
	dirs, err := dirFile.Readdir(-1)
	if err != nil {
		panic(err)
	}
	for _, info := range dirs {
		if info.IsDir() {
			parseDir(filepath.Join(dir, info.Name()))
		}
	}
	//}

}

func printTypeLine(w io.Writer, pkg *ast.Package, kind ast.ObjKind, format string) {
	names := []string{}
	//otherNamesAndTypes := []
	for _, f := range pkg.Files {
		for name, object := range f.Scope.Objects {
			if object.Kind == kind && (ast.IsExported(name)) {
				isStruct := reflect.TypeOf(object.Decl.(*ast.TypeSpec).Type) == reflect.TypeOf(&ast.StructType{})
				if isStruct {
					// currently we only list plain structs
					names = append(names, name)
				} else {
					fmt.Printf("%s, %#v\n", name, object.Decl.(*ast.TypeSpec).Type)
				}
			}

		}
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, format, name)
	}
}

func printEnumLine(w io.Writer, pkg *ast.Package, kind ast.ObjKind, format string) {
	names := []string{}
	for _, f := range pkg.Files {
		for name, object := range f.Scope.Objects {
			if object.Kind == kind && (ast.IsExported(name)) {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, format, name, name)
	}
}

func filter(info os.FileInfo) bool {

	name := info.Name()

	if info.IsDir() {
		return false
	}

	if name == gofile {
		return false
	}

	if filepath.Ext(name) != ".go" {
		return false
	}

	if strings.HasSuffix(name, "_test.go") {
		return false
	}

	return true

}
