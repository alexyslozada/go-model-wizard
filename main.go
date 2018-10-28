package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	strcase "github.com/stoewer/go-strcase"
)

var (
	tpl  *template.Template
	fm   = template.FuncMap{}
	n    string
	t    string
	h    Helper
	dest string
)

func init() {
	setHelpers()
	tpl = template.Must(template.New("").Funcs(fm).ParseGlob("templates/*.gotpl"))
}

// Model estructura que se generará
type Model struct {
	Name          string
	Table         string
	Fields        []Field
	PackageRoutes map[string]string
}

// Field structura de un tipo de campo del modelo
type Field struct {
	Name    string
	Type    string
	NotNull string
}

// Helper estructura que permite procesar los campos digitados
type Helper struct {
	Fields []Field
}

// String permite imprimir los campos
func (h *Helper) String() string {
	return fmt.Sprint(h.Fields)
}

// Set permite distribuir los campos escritos en el flag
func (h *Helper) Set(value string) error {
	fields := strings.Split(value, ",")
	for _, v := range fields {
		field := strings.Split(v, ":")
		nn := "NOT NULL"
		if len(field) == 3 {
			if strings.ToLower(field[2]) == "f" {
				nn = ""
			}
		}
		f := Field{field[0], field[1], nn}
		h.Fields = append(h.Fields, f)
	}
	return nil
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
				return "VARCHAR(SIZE)"
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

func main() {
	// Ruta a los paquetes de configuracion, logger, mensajes, module_role
	cnfg := ""
	logg := ""
	mess := ""
	modr := ""

	flag.StringVar(&n, "model", "", "nombre del modelo (ej: role)")
	flag.StringVar(&t, "table", "", "nombre de la tabla (ej: roles)")
	flag.Var(&h, "fields", "nombre de los campos de la tabla y su tipo, separados por coma sin espacios (ej: name:string,phone:string,address:string,age:int)")
	flag.StringVar(&dest, "dest", "", "destino de los archivos a crear. siempre se creará después de $GOPATH/src/. Es decir si se coloca github.com/alexys/miproyecto se crearán en $GOPATH/src/github.com/alexys/miproyecto")
	flag.StringVar(&cnfg, "cnfg", "", "ruta del paquete de configuracion")
	flag.StringVar(&logg, "logg", "", "ruta del paquete de logger")
	flag.StringVar(&mess, "mess", "", "ruta del paquete de mensajes")
	flag.StringVar(&modr, "modr", "", "ruta del paquete de modulo por role")
	flag.Parse()

	if n == "" || t == "" || len(h.Fields) == 0 ||
		dest == "" || cnfg == "" || logg == "" ||
		mess == "" || modr == "" {
		flag.Usage()
		log.Fatalln("todos los flag son obligatorios")
	}

	ps := make(map[string]string)
	ps["configuration"] = cnfg
	ps["logger"] = logg
	ps["message"] = mess
	ps["module_role"] = modr

	m := Model{n, t, h.Fields, ps}

	gopath := os.Getenv("GOPATH")
	realDest := []string{gopath, "src"}
	realDest = append(realDest, strings.Split(dest, "/")...)
	gp := filepath.Join(realDest...)
	pks := filepath.Join(gp, "models")
	ds := filepath.Join(gp, "database")

	pk := filepath.Join(pks, n)

	createDir(pk)
	createDir(ds)

	generateSQL(m, ds)
	generateModel(m, pk)
	generateStorage(m, pk)
	generatePsql(m, pk)
	generateHandler(m, pk)
	generateRoute(m, pk)
}

// createDir crea el directorio de destino de los archivos
func createDir(d string) {
	_, err := os.Stat(d)
	if os.IsNotExist(err) {
		log.Printf("no existe la carpeta %s. Creandola...", d)
		os.MkdirAll(d, os.ModePerm)
	}
}

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

func formatFile(filePath string) {
	cmd := exec.Command("gofmt", "-w", filePath)
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: No se pudo ejecutar gofmt")
	}
}
