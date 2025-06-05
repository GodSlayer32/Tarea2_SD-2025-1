package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "google.golang.org/grpc"
    pb "tarea_sd/proto/gen/emergencia"
)

// Define la estructura para leer JSON de emergencias
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

    //Lee los datos del JSON de emegencia
    data, err := os.ReadFile(os.Args[1])
    if err != nil {
        log.Fatalf("No se pudo leer el archivo: %v", err)
    }

    var emergencias []Emergencia
    if err := json.Unmarshal(data, &emergencias); err != nil {
        log.Fatalf("Error al parsear JSON: %v", err)
    }

    fmt.Println("Emergencias leídas correctamente")

    //Aquí realiza la conexión gRPC con el servicio de Asignación desde el Cliente
    connAsig, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("No se pudo conectar a asignación: %v", err)
    }
    defer connAsig.Close()


    // Creación del cliente gRPC para el servicio de asignación
    asignacionClient := pb.NewServicioAsignacionClient(connAsig)
    
    stream, err := asignacionClient.EnviarEmergencias(context.Background())
    if err != nil {
        log.Fatalf("Error al abrir stream con asignación: %v", err)
    }

    //Aquí realiza envío de JSON de emergencias al server
    for _, e := range emergencias {
        fmt.Printf("Enviando emergencia: %s\n", e.Name)
        err := stream.Send(&pb.Emergencia{
            Name:      e.Name,
            Latitude:  e.Latitude,
            Longitude: e.Longitude,
            Magnitude: e.Magnitude,
        })
        if err != nil {
            log.Fatalf("Error al enviar emergencia: %v", err)
        }
    }

    _, err = stream.CloseAndRecv()
    if err != nil {
        log.Fatalf("Error al cerrar stream: %v", err)
    }

    fmt.Println("Todas las emergencias han sido enviadas")

    //Aquí realiza la conexión gRPC con el servicio de Monitoreo 
    connMon, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("No se pudo conectar a monitoreo: %v", err)
    }
    defer connMon.Close()

    // Creación del cliente gRPC para el servicio de monitoreo
    monitoreoClient := pb.NewServicioMonitoreoClient(connMon)

    statusStream, err := monitoreoClient.RecibirActualizaciones(context.Background(), &pb.Vacio{})
    if err != nil {
        log.Fatalf("Error al recibir actualizaciones: %v", err)
    }

    fmt.Println("Escuchando actualizaciones de estado...")

    // Recibe estados continuamente y los muestra en tiempo real
    for {
        estado, err := statusStream.Recv()
        if err != nil {
            log.Fatalf("Error recibiendo estado: %v", err)
        }
        fmt.Printf(" %s — Estado: %s — Dron: %s\n", estado.Name, estado.Status, estado.DronId)
    }
}
