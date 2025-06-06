package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	pb "tarea_sd/proto/gen/emergencia"

	"google.golang.org/grpc"
)

type Emergencia struct {
	Name      string `json:"name"`
	Latitude  int32  `json:"latitude"`
	Longitude int32  `json:"longitude"`
	Magnitude int32  `json:"magnitude"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Uso: ./cliente emergencia.json")
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("No se pudo leer el archivo: %v", err)
	}

	var emergencias []Emergencia
	if err := json.Unmarshal(data, &emergencias); err != nil {
		log.Fatalf("Error al parsear JSON: %v", err)
	}

	fmt.Println("Emergencias leídas correctamente")

	// gRPC con monitoreo
	connMon, err := grpc.Dial("10.10.28.26:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No se pudo conectar a monitoreo: %v", err)
	}
	defer connMon.Close()
	monitoreoClient := pb.NewServicioMonitoreoClient(connMon)

	// gRPC con asignación
	connAsig, err := grpc.Dial("10.10.28.27:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No se pudo conectar a asignación: %v", err)
	}
	defer connAsig.Close()
	asignacionClient := pb.NewServicioAsignacionClient(connAsig)

	// Conexión gRPC persistente al monitoreo (stream abierto una vez)
	statusStream, err := monitoreoClient.RecibirActualizaciones(context.Background(), &pb.Vacio{})
	if err != nil {
		log.Fatalf("Error recibiendo actualizaciones: %v", err)
	}

	// Canal para recibir estados desde goroutine
	estadoChan := make(chan *pb.EstadoEmergencia)

	go func() {
		for {
			estado, err := statusStream.Recv()
			if err != nil {
				log.Fatalf("Error recibiendo estado: %v", err)
			}
			estadoChan <- estado
		}
	}()

	// Envío y espera de emergencias
	for _, e := range emergencias {
		fmt.Printf("\n Emergencia actual: %s magnitud %d en x = %d , y = %d\n", e.Name, e.Magnitude, e.Latitude, e.Longitude)

		// Envío por stream
		stream, err := asignacionClient.EnviarEmergencias(context.Background())
		if err != nil {
			log.Fatalf("Error abriendo stream a asignación: %v", err)
		}

		err = stream.Send(&pb.Emergencia{
			Name:      e.Name,
			Latitude:  e.Latitude,
			Longitude: e.Longitude,
			Magnitude: e.Magnitude,
		})
		if err != nil {
			log.Fatalf("Error enviando emergencia: %v", err)
		}

		_, err = stream.CloseAndRecv()
		if err != nil {
			log.Fatalf("Error cerrando stream de asignación: %v", err)
		}

		asignado := false
		for {
			estado := <-estadoChan
			if estado.Name == e.Name {
				if !asignado && strings.Contains(estado.Status, "Dron en camino") {
					fmt.Printf(" Se ha asignado %s a la emergencia\n", estado.DronId)
					asignado = true
				}

				fmt.Printf(" %s — Estado: %s — Dron: %s\n", estado.Name, estado.Status, estado.DronId)

				if strings.Contains(strings.ToLower(estado.Status), "extinguido") {
					break
				}
			}
		}
	}

	fmt.Println("\n Todas las emergencias han sido atendidas.")
}
