//go:build ignore

// check-modules.go — verify MODULES.md claimed counts against actual codebase
// Usage: go run scripts/check-modules.go
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Result struct {
	Name    string
	Claimed int
	Actual  int
}

func main() {
	// Find repo root (directory containing MODULES.md)
	repoRoot := findRepoRoot()

	// Parse claimed counts from MODULES.md
	claimed := parseClaimedCounts(filepath.Join(repoRoot, "MODULES.md"))

	fmt.Println("=== MODULES.md Sync Check ===")
	fmt.Printf("Claimed: model=%d, handler=%d, service=%d, repository=%d\n\n",
		claimed["model"], claimed["handler"], claimed["service"], claimed["repository"])

	// Count actual types using AST
	actualModels := countModelTypes(filepath.Join(repoRoot, "internal", "model"))
	actualHandlers := countStructsWithSuffix(filepath.Join(repoRoot, "internal", "handler"), "Handler")
	actualServices := countStructsWithSuffix(filepath.Join(repoRoot, "internal", "service"), "Service")
	actualRepos := countStructsWithSuffix(filepath.Join(repoRoot, "internal", "repository"), "Repository")
	actualMigrations := countMigrationFiles(filepath.Join(repoRoot, "internal", "pkg", "dbmigrate", "migrations"))

	results := []Result{
		{"model types (TableName)", claimed["model"], actualModels},
		{"handler structs (*Handler)", claimed["handler"], actualHandlers},
		{"service structs (*Service)", claimed["service"], actualServices},
		{"repository structs (*Repository)", claimed["repository"], actualRepos},
	}

	pass, fail := 0, 0
	for _, r := range results {
		if r.Claimed == r.Actual {
			fmt.Printf("  PASS  %s: claimed=%d actual=%d\n", r.Name, r.Claimed, r.Actual)
			pass++
		} else {
			fmt.Printf("  FAIL  %s: claimed=%d actual=%d\n", r.Name, r.Claimed, r.Actual)
			fail++
		}
	}

	fmt.Printf("\n  INFO  migration .up.sql files: %d\n\n", actualMigrations)
	fmt.Printf("=== Result: %d passed, %d failed ===\n", pass, fail)

	if fail > 0 {
		fmt.Println("MODULES.md is out of sync with the actual codebase.")
		fmt.Println("Update the counts in MODULES.md header: > 共 N 个 model, N 个 handler, N 个 service, N 个 repository")
		os.Exit(1)
	}

	fmt.Println("MODULES.md is in sync.")
}

// findRepoRoot walks up from cwd to find a directory containing MODULES.md
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot get working directory: %v\n", err)
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "MODULES.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintf(os.Stderr, "ERROR: cannot find MODULES.md in any parent directory\n")
			os.Exit(1)
		}
		dir = parent
	}
}

// parseClaimedCounts extracts "共 N 个 model, N 个 handler, N 个 service, N 个 repository"
func parseClaimedCounts(path string) map[string]int {
	result := map[string]int{"model": 0, "handler": 0, "service": 0, "repository": 0}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot open %s: %v\n", path, err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	re := regexp.MustCompile(`(\d+)\s+个\s+(model|handler|service|repository)`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "个 model") && strings.Contains(line, "个 handler") {
			matches := re.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) == 3 {
					n, _ := strconv.Atoi(m[1])
					result[m[2]] = n
				}
			}
			break
		}
	}
	return result
}

// countModelTypes counts struct types in the model package that have TableName() methods.
// This is the most accurate way to count "DB models".
func countModelTypes(dir string) int {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: cannot parse %s: %v\n", dir, err)
		return 0
	}

	// Collect all struct type names and all receiver types with TableName()
	structNames := make(map[string]bool)
	tableNameTypes := make(map[string]bool)

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					// type Foo struct { ... }
					if d.Tok == token.TYPE {
						for _, spec := range d.Specs {
							if ts, ok := spec.(*ast.TypeSpec); ok {
								if _, ok := ts.Type.(*ast.StructType); ok {
									structNames[ts.Name.Name] = true
								}
							}
						}
					}
				case *ast.FuncDecl:
					// func (Foo) TableName() string { ... }
					if d.Recv != nil && d.Name.Name == "TableName" {
						if recvType := receiverTypeName(d.Recv); recvType != "" {
							tableNameTypes[recvType] = true
						}
					}
				}
			}
		}
	}

	// Return count of structs that have TableName()
	count := 0
	for name := range tableNameTypes {
		if structNames[name] {
			count++
		}
	}
	return count
}

// countStructsWithSuffix counts exported struct types whose name ends with the given suffix
// (e.g., "Handler", "Service", "Repository") in a directory, excluding test files.
func countStructsWithSuffix(dir string, suffix string) int {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: cannot parse %s: %v\n", dir, err)
		return 0
	}

	counted := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, ok := ts.Type.(*ast.StructType); ok {
						if strings.HasSuffix(ts.Name.Name, suffix) {
							counted[ts.Name.Name] = true
						}
					}
				}
			}
		}
	}
	return len(counted)
}

// countMigrationFiles counts *.up.sql files in the migrations directory
func countMigrationFiles(dir string) int {
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: cannot read %s: %v\n", dir, err)
		return 0
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			count++
		}
	}
	return count
}

// receiverTypeName extracts the type name from an AST receiver field list
func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	ft := recv.List[0].Type
	// Handle pointer receivers (*Foo)
	if star, ok := ft.(*ast.StarExpr); ok {
		ft = star.X
	}
	if ident, ok := ft.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}
