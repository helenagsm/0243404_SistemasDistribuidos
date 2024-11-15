# Etapa de construcción
FROM golang:latest

# Crea el directorio y luego establece el directorio de trabajo
RUN mkdir -p /app
WORKDIR /app


COPY . .
# Copia solo los archivos de dependencias primero (para optimización de caché)
RUN go mod download

# Copia todo el proyecto en el contenedor


# Construye el binario
RUN go build -o test-runner

# Comando para ejecutar el binario
CMD ["./test-runner"]
