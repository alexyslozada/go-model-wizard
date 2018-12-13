package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/fatih/color"
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
	// rutas de los paquetes de configuración, logger, message, model_role
	ps map[string]string
)

func init() {
	setHelpers()
	tpl = template.Must(template.New("").Funcs(fm).ParseGlob("templates/*.gotpl"))
}

func main() {
	ff := flag.String("file", "", "utiliza este flag si deseas generar paquetes desde un archivo.")
	fc := flag.String("config", "./config.json", "ubicación del archivo json de configuración.")
	flag.Parse()

	readConfigFile(*fc)

	color.Green("Iniciando proceso...")

	if *ff != "" {
		generateFromFile(*ff)
	} else {
		showMainMenu()
		execute()
	}

	color.Green("Proceso finalizado.")
}

func execute() {
	m := Model{n, t, fs, ps}
	gopath := os.Getenv("GOPATH")
	realDest := []string{gopath, "src"}
	realDest = append(realDest, strings.Split(ps["dest"], "/")...)
	gp := filepath.Join(realDest...)
	pks := filepath.Join(gp, ps["packages_folder"])
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

func formatFile(filePath string) {
	cmd := exec.Command("gofmt", "-w", filePath)
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: No se pudo ejecutar gofmt")
	}
}

func readConfigFile(cf string) {
	ps = make(map[string]string, 0)

	file, err := ioutil.ReadFile(cf)
	if err != nil {
		e := fmt.Sprintf("no se pudo abrir el archivo de configuración: %v", err)
		color.Red(e)
		os.Exit(1)
	}

	err = json.Unmarshal(file, &ps)
	if err != nil {
		e := fmt.Sprintf("no se pudo convertir la configuración en mapa: %v", err)
		color.Red(e)
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
