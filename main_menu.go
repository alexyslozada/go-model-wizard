package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/fatih/color"
)

func showMainMenu() {
	color.Cyan("*************************************")
	color.Cyan("* Sistema de generación de paquetes *")
	color.Cyan("*************************************")
	fmt.Println()

	scanName()
	scanTable()
	scanFields()
}

func scanName() {
	color.Cyan("1. Digite el nombre del paquete en singular y minuscula:")
	fmt.Scan(&n)
	if n == "" {
		color.Red("el nombre del paquete es obligatorio")
		os.Exit(1)
	}
}

func scanTable() {
	color.Cyan("2. Digite el nombre de la tabla en plural y minuscula:")
	fmt.Scan(&t)
	if t == "" {
		color.Red("el nombre de la tabla es obligatorio")
		os.Exit(1)
	}
}

func scanFields() {
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
}
