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

// Canal para compartir mensajes desde RabbitMQ al gRPC
var updatesChan = make(chan *pb.EstadoEmergencia, 100)

// Conexión a RabbitMQ y escucha de la cola "monitoreo"
func iniciarRabbitMQ() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
    }

    ch, err := conn.Channel()
    if err != nil {
        log.Fatalf("❌ Error creando canal: %v", err)
    }

    _, err = ch.QueueDeclare("monitoreo", false, false, false, false, nil)
    if err != nil {
        log.Fatalf("❌ Error declarando cola monitoreo: %v", err)
    }

    msgs, err := ch.Consume(
        "monitoreo", "", true, false, false, false, nil,
    )
    if err != nil {
        log.Fatalf("❌ Error consumiendo cola: %v", err)
    }

    // Procesar mensajes
    go func() {
        for msg := range msgs {
            var data map[string]string
            if err := json.Unmarshal(msg.Body, &data); err != nil {
                log.Printf("⚠️  Error parseando JSON: %v", err)
                continue
            }

            update := &pb.EstadoEmergencia{
                Name:   data["name"],
                Status: data["status"],
                DronId: data["dron_id"],
            }

            log.Printf("📥 Recibido de RabbitMQ: %v", update)
            updatesChan <- update
        }
    }()
}

// Servidor gRPC de monitoreo
type servidorMonitoreo struct {
    pb.UnimplementedServicioMonitoreoServer
}

func (s *servidorMonitoreo) RecibirActualizaciones(_ *pb.Vacio, stream pb.ServicioMonitoreo_RecibirActualizacionesServer) error {
    log.Println("🔗 Cliente conectado a monitoreo")

    for update := range updatesChan {
        if err := stream.Send(update); err != nil {
            log.Printf("❌ Error enviando al cliente: %v", err)
            return err
        }
    }

    return nil
}

func main() {
    iniciarRabbitMQ()

    lis, err := net.Listen("tcp", ":50052")
    if err != nil {
        log.Fatalf("❌ No se pudo iniciar servidor gRPC: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterServicioMonitoreoServer(grpcServer, &servidorMonitoreo{})

    fmt.Println("📡 Servicio de monitoreo escuchando en puerto 50052")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("❌ Error al servir gRPC: %v", err)
    }
}
