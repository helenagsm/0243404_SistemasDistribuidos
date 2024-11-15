package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	// Obtiene la ruta absoluta del proyecto
	pathOfProject, err := filepath.Abs(".")
	if err != nil {
		panic(fmt.Sprintf("Error obteniendo la ruta del proyecto: %v", err))
	}

	// Define la ruta específica a la carpeta 'server'
	pathToServer := filepath.Join(pathOfProject, "server")

	for {
		// Ejecuta `go test` en la carpeta `server`
		cmd := exec.Command("go", "test", pathToServer)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error ejecutando pruebas: %v\n", err)
		}
		fmt.Println(string(output))

		// Espera un minuto antes de repetir las pruebas
		time.Sleep(1 * time.Minute)
	}
}
