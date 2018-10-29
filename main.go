package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/color"

	strcase "github.com/stoewer/go-strcase"
)

var (
	tpl *template.Template
	fm  = template.FuncMap{}
	// n nombre del paqute
	n string
	// t nombre de la tabla
	t string
	// fs los campos del modelo
	fs []Field
	// dest destino del paquete
	dest string
	// rutas de los paquetes de configuración, logger, message, model_role
	ps map[string]string
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
	Len     int
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

func main() {
	showHeader()

	color.Green("Iniciando proceso...")

	m := Model{n, t, fs, ps}

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

	color.Green("Proceso finalizado.")
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

func showHeader() {
	color.Cyan("*************************************")
	color.Cyan("* Sistema de generación de paquetes *")
	color.Cyan("*************************************")
	fmt.Println()
	color.Cyan("1. Digite el nombre del paquete en singular y minuscula:")
	fmt.Scan(&n)
	if n == "" {
		color.Red("el nombre del paquete es obligatorio")
		os.Exit(1)
	}
	color.Cyan("2. Digite el nombre de la tabla en plural y minuscula:")
	fmt.Scan(&t)
	if t == "" {
		color.Red("el nombre de la tabla es obligatorio")
		os.Exit(1)
	}

	color.Cyan("3. Digite los campos del modelo.")
	color.Cyan("El formato es: nombre:tipo:nonulo:tamaño.")
	color.Cyan("* cada campo debe estar separada por un espacio. ej:")
	color.Cyan("name:string:f:50 age:int birth:time.Time:t other:bool")
	color.Cyan("* nombre: nombre del campo, minúsculas.")
	color.Cyan("* tipo: string, int, float32, float64, time.Time, bool.")
	color.Cyan("* nonulo: t si permite nulos, f no permite nulos. (por defecto es f)")
	color.Cyan("* tamaño: número entero. Sólo aplica para string.")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	fields := scanner.Text()
	err := scanner.Err()
	if err != nil {
		color.Red("error al leer los campos:", err)
		os.Exit(1)
	}

	fs = getFields(fields)
	if len(fs) == 0 {
		color.Red("no se han recibido campos del modelo")
		os.Exit(1)
	}

	color.Cyan("4. Destino del paquete")
	color.Cyan("* se debe colocar la ruta del destino sin $GOPATH/src/")
	color.Cyan("* ej: github.com/alexyslozada/miproyecto/modelos")
	fmt.Scan(&dest)
	if dest == "" {
		color.Red("el destino es obligatorio")
		os.Exit(1)
	}

	ps = make(map[string]string)
	color.Cyan("5. Ubicación del paquete de configuracion")
	color.Cyan("* se debe colocar sin $GOPATH/src/")
	v := ""
	fmt.Scan(&v)
	ps["configuration"] = v
	if ps["configuration"] == "" {
		color.Red("la ubicación del paquete es obligatorio")
		os.Exit(1)
	}

	color.Cyan("6. Ubicación del paquete de logger")
	color.Cyan("* se debe colocar sin $GOPATH/src/")
	color.Cyan("* si es la misma ruta de configuration, coloque el signo igual: =")
	fmt.Scan(&v)
	if strings.TrimSpace(v) == "=" {
		ps["logger"] = ps["configuration"]
	} else {
		ps["logger"] = v
	}

	if ps["logger"] == "" {
		color.Red("la ubicación del paquete es obligatorio")
		os.Exit(1)
	}

	color.Cyan("7. Ubicación del paquete de mensajes")
	color.Cyan("* se debe colocar sin $GOPATH/src/")
	color.Cyan("* si es la misma ruta de configuration, coloque el signo igual: =")
	fmt.Scan(&v)
	if strings.TrimSpace(v) == "=" {
		ps["message"] = ps["configuration"]
	} else {
		ps["message"] = v
	}
	if ps["message"] == "" {
		color.Red("la ubicación del paquete es obligatorio")
		os.Exit(1)
	}

	color.Cyan("8. Ubicación del paquete de roles por módulo")
	color.Cyan("* se debe colocar sin $GOPATH/src/")
	color.Cyan("* si es la misma ruta de configuration, coloque el signo igual: =")
	fmt.Scan(&v)
	if strings.TrimSpace(v) == "=" {
		ps["module_role"] = ps["configuration"]
	} else {
		ps["module_role"] = v
	}
	if ps["module_role"] == "" {
		color.Red("la ubicación del paquete es obligatorio")
		os.Exit(1)
	}

	color.Cyan("9. Ubicación del paquete de login")
	color.Cyan("* se debe colocar sin $GOPATH/src/")
	color.Cyan("* si es la misma ruta de configuration, coloque el signo igual: =")
	fmt.Scan(&v)
	if strings.TrimSpace(v) == "=" {
		ps["login"] = ps["configuration"]
	} else {
		ps["login"] = v
	}
	if ps["login"] == "" {
		color.Red("la ubicación del paquete es obligatorio")
		os.Exit(1)
	}

	color.Cyan("10. Ubicación del paquete de psql (utilidades de sql)")
	color.Cyan("* se debe colocar sin $GOPATH/src/")
	color.Cyan("* si es la misma ruta de configuration, coloque el signo igual: =")
	fmt.Scan(&v)
	if strings.TrimSpace(v) == "=" {
		ps["psql"] = ps["configuration"]
	} else {
		ps["psql"] = v
	}
	if ps["psql"] == "" {
		color.Red("la ubicación del paquete es obligatorio")
		os.Exit(1)
	}
}

func getFields(value string) []Field {
	var err error
	rs := make([]Field, 0)
	fields := strings.Split(value, " ")
	for _, v := range fields {
		field := strings.Split(v, ":")
		nn := "NOT NULL"
		i := 0
		if len(field) >= 3 {
			if strings.ToLower(field[2]) == "t" {
				nn = ""
			}
		}
		if len(field) == 4 {
			i, err = strconv.Atoi(field[3])
			if err != nil {
				log.Fatalf("%s no es un número válido: %v", field[3], err)
			}

		}
		f := Field{field[0], field[1], nn, i}
		rs = append(rs, f)
	}

	return rs
}
