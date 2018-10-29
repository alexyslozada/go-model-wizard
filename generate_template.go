package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	strcase "github.com/stoewer/go-strcase"
)

// generateSQL crea el archivo sql
func generateSQL(m Model, d string) {
	now := time.Now()
	fn := now.Format("20060102") + "_" + now.Format("150405") + "_create_" + m.Table + ".sql"
	generateTemplate(filepath.Join(d, fn), "table.gotpl", m)
}

// generateModel crea el modelo
func generateModel(m Model, d string) {
	generateTemplate(filepath.Join(d, "model.go"), "model.gotpl", m)
}

// generateStorage crea la interface storage
func generateStorage(m Model, d string) {
	generateTemplate(filepath.Join(d, "storage.go"), "storage.gotpl", m)
}

// generatePsql crea el archivo psql
func generatePsql(m Model, d string) {
	generateTemplate(filepath.Join(d, "psql.go"), "psql.gotpl", m)
}

// generateHandler crea el handler
func generateHandler(m Model, d string) {
	generateTemplate(filepath.Join(d, "handler.go"), "handler.gotpl", m)
}

// generateRoute crea el route
func generateRoute(m Model, d string) {
	generateTemplate(filepath.Join(d, "route.go"), "router.gotpl", m)
}

// generateTemplate crea el archivo .go con base al template
func generateTemplate(dest, source string, m Model) {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("no se pudo crear el archivo: %v", err)
	}
	if filepath.Ext(dest) == ".go" {
		defer formatFile(dest)
	}
	defer f.Close()

	err = tpl.ExecuteTemplate(f, source, m)
	if err != nil {
		log.Printf("error creando el archivo: %v", err)
		return
	}
}

func setHelpers() {
	fm = template.FuncMap{
		"ucc": func(v string) string {
			return strcase.UpperCamelCase(v)
		},
		"upp": func(v string) string {
			return strings.ToUpper(v)
		},
		"kcc": func(v string) string {
			return strcase.KebabCase(v)
		},
		"lcc": func(v string) string {
			return strcase.LowerCamelCase(v)
		},
		"inc": func(v int) int {
			return v + 1
		},
		"dec": func(v int) int {
			return v - 1
		},
		"sqlType": func(v string) string {
			switch v {
			case "uint":
				fallthrough
			case "int":
				return "INT"
			case "string":
				return "VARCHAR"
			case "bool":
				return "BOOLEAN"
			case "time.Time":
				return "TIMESTAMP"
			}
			return "CHANGE-THIS-TYPE"
		},
		"fieldSQL": func(f Field) string {
			field := strcase.UpperCamelCase(f.Name)

			if f.NotNull == "NOT NULL" {
				return fmt.Sprintf("m.%s", field)
			}

			switch f.Type {
			case "string":
				return fmt.Sprintf("psql.StringToNull(m.%s)", field)
			case "int":
				fallthrough
			case "uint":
				return fmt.Sprintf("psql.IntToNull(int64(m.%s))", field)
			case "time.Time":
				return fmt.Sprintf("psql.TimeToNull(m.%s)", field)
			default:
				return fmt.Sprintf("Error: no existe el tipo de dato: %s", t)
			}
		},
		"fieldSQLScan": func(f Field) string {
			if f.NotNull == "NOT NULL" {
				return ""
			}

			switch f.Type {
			case "string":
				return fmt.Sprintf("%s := sql.NullString{}", f.Name)
			case "int":
				fallthrough
			case "uint":
				return fmt.Sprintf("%s := sql.NullInt64{}", f.Name)
			case "time.Time":
				return fmt.Sprintf("%s := pq.NullTime{}", f.Name)
			case "bool":
				return fmt.Sprintf("%s := sql.NullBool{}", f.Name)
			default:
				return fmt.Sprintf("Error: no existe el tipo de dato: %s", t)
			}
		},
		"fieldSQLScanValue": func(f Field) string {
			field := strcase.UpperCamelCase(f.Name)
			if f.NotNull == "NOT NULL" {
				return ""
			}

			switch f.Type {
			case "string":
				return fmt.Sprintf("m.%s = %s.String", field, f.Name)
			case "int":
				fallthrough
			case "uint":
				return fmt.Sprintf("m.%s = %s(%s.Int64)", field, f.Type, f.Name)
			case "time.Time":
				return fmt.Sprintf("m.%s = %s.Time", field, f.Name)
			case "bool":
				return fmt.Sprintf("m.%s = %s.Bool", field, f.Name)
			default:
				return fmt.Sprintf("Error: no existe el tipo de dato: %s", t)
			}
		},
	}
}
