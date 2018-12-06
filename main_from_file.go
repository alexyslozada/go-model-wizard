package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

func generateFromFile(file string) {
	isCSV := strings.HasSuffix(file, ".csv")

	if !isCSV {
		color.Red("el archivo de importación de paquetes debe tener extensión .csv")
		os.Exit(1)
	}

	f, err := os.Open(file)
	if err != nil {
		color.Red(fmt.Sprintf("no se pudo abrir el archivo de importación de paquetes: %v", err))
		os.Exit(1)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = 3

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			color.Red(fmt.Sprintf("error leyendo la línea del archivo de importación de paquetes: %v", err))
			os.Exit(1)
		}

		if record[2] == "" {
			color.Red(fmt.Sprintf("no se procesó el modelo: %s porque no se recibieron campos", record[0]))
			continue
		}

		n = record[0]
		t = record[1]
		fs = getFields(record[2])

		execute()
	}
}
