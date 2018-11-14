package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	showMainMenu()

	color.Green("Iniciando proceso...")

	m := Model{n, t, fs, ps}
	gopath := os.Getenv("GOPATH")
	realDest := []string{gopath, "src"}
	realDest = append(realDest, strings.Split(ps["dest"], "/")...)
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

func formatFile(filePath string) {
	cmd := exec.Command("gofmt", "-w", filePath)
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: No se pudo ejecutar gofmt")
	}
}
