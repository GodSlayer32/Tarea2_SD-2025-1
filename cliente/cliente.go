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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	// Leer archivo de emergencias
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("No se pudo leer el archivo: %v", err)
	}

	var emergencias []Emergencia
	if err := json.Unmarshal(data, &emergencias); err != nil {
		log.Fatalf("Error al parsear JSON: %v", err)
	}

	fmt.Println("Emergencias leídas correctamente")

	// Contexto para monitoreo, cancelable
	ctxMon, cancelMon := context.WithCancel(context.Background())
	defer cancelMon()

	// Conexión gRPC a Monitoreo
	connMon, err := grpc.Dial("10.10.28.26:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No se pudo conectar a monitoreo: %v", err)
	}
	defer connMon.Close()
	monClient := pb.NewServicioMonitoreoClient(connMon)

	// Conexión gRPC a Asignación
	connAsig, err := grpc.Dial("10.10.28.27:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No se pudo conectar a asignación: %v", err)
	}
	defer connAsig.Close()
	asigClient := pb.NewServicioAsignacionClient(connAsig)

	// Iniciar stream de Monitoreo
	statusStream, err := monClient.RecibirActualizaciones(ctxMon, &pb.Vacio{})
	if err != nil {
		log.Fatalf("Error iniciando stream de monitoreo: %v", err)
	}

	// Canal para estados
	estadoChan := make(chan *pb.EstadoEmergencia)

	// Goroutine de recepción de estados
	go func() {
		for {
			estado, err := statusStream.Recv()
			if err != nil {
				// Fin del stream o error de conexión
				if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
					log.Println("Monitoreo cancelado por contexto")
				} else {
					log.Printf("Error recibiendo estado: %v", err)
				}
				break
			}
			estadoChan <- estado
		}
		close(estadoChan)
	}()

	// Procesar emergencias
	for _, e := range emergencias {
		fmt.Printf("\nEmergencia actual: %s magnitud %d en x = %d, y = %d\n", e.Name, e.Magnitude, e.Latitude, e.Longitude)

		// Enviar emergencia a Asignación
		stream, err := asigClient.EnviarEmergencias(context.Background())
		if err != nil {
			log.Fatalf("Error abriendo stream de asignación: %v", err)
		}
		if err := stream.Send(&pb.Emergencia{Name: e.Name, Latitude: e.Latitude, Longitude: e.Longitude, Magnitude: e.Magnitude}); err != nil {
			log.Fatalf("Error enviando emergencia: %v", err)
		}
		if _, err := stream.CloseAndRecv(); err != nil {
			log.Fatalf("Error cerrando stream de asignación: %v", err)
		}

		// Mostrar estados hasta extinción
		asignado := false
		for estado := range estadoChan {
			if estado.Name != e.Name {
				continue
			}
			if !asignado && strings.Contains(estado.Status, "Dron en camino") {
				fmt.Printf("Se ha asignado %s a la emergencia\n", estado.DronId)
				asignado = true
			}

			fmt.Printf("%s — Estado: %s — Dron: %s\n", estado.Name, estado.Status, estado.DronId)
			if strings.Contains(strings.ToLower(estado.Status), "extinguido") {
				break
			}
		}
	}

	fmt.Println("\n✅ Todas las emergencias han sido atendidas.")
}
