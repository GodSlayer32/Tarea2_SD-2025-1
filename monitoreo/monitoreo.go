package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net"

    amqp "github.com/rabbitmq/amqp091-go"
    "google.golang.org/grpc"
    pb "tarea_sd/proto/gen/emergencia"
)

// Implementa el servicio gRPC de monitoreo
type servidorMonitoreo struct {
    pb.UnimplementedServicioMonitoreoServer
}

// Cada cliente conectado crea su propio stream desde RabbitMQ
func (s *servidorMonitoreo) RecibirActualizaciones(_ *pb.Vacio, stream pb.ServicioMonitoreo_RecibirActualizacionesServer) error {
    log.Println("Cliente conectado a monitoreo")

    // Conexión individual a RabbitMQ por cliente
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Printf("Error conectando a RabbitMQ: %v", err)
        return err
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Printf("Error creando canal RabbitMQ: %v", err)
        return err
    }
    defer ch.Close()

    _, err = ch.QueueDeclare("monitoreo", false, false, false, false, nil)
    if err != nil {
        log.Printf("Error declarando cola monitoreo: %v", err)
        return err
    }

    msgs, err := ch.Consume("monitoreo", "", true, false, false, false, nil)
    if err != nil {
        log.Printf("Error iniciando consumo de RabbitMQ: %v", err)
        return err
    }

    // Recibir y enviar en tiempo real
    for msg := range msgs {
        var data map[string]string
        if err := json.Unmarshal(msg.Body, &data); err != nil {
            log.Printf("Error parseando mensaje: %v", err)
            continue
        }

        update := &pb.EstadoEmergencia{
            Name:   data["name"],
            Status: data["status"],
            DronId: data["dron_id"],
        }

        log.Printf("Estado recibido: %+v", update)

        if err := stream.Send(update); err != nil {
            log.Printf("Error enviando al cliente: %v", err)
            return err
        }
    }

    return nil
}

func main() {
    lis, err := net.Listen("tcp", ":50052")
    if err != nil {
        log.Fatalf("No se pudo iniciar servidor gRPC: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterServicioMonitoreoServer(grpcServer, &servidorMonitoreo{})

    fmt.Println("Servicio de monitoreo escuchando en puerto 50052")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Error al servir gRPC: %v", err)
    }
}
