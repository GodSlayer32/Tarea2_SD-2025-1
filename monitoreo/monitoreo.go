package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	pb "tarea_sd/proto/gen/emergencia"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// Gestiona suscriptores para broadcasting de actualizaciones
var (
	subsMu      sync.Mutex
	subscribers = make(map[int]pb.ServicioMonitoreo_RecibirActualizacionesServer)
	nextSubID   = 0
)

// Conexión a RabbitMQ y broadcast de mensajes a todos los suscriptores
func iniciarRabbitMQ() {
	conn, err := amqp.Dial("amqp://tarea:tarea123@10.10.28.27:5672/")
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creando canal RabbitMQ: %v", err)
	}

	// Asegurar existencia de cola
	if _, err := ch.QueueDeclare("monitoreo", false, false, false, false, nil); err != nil {
		log.Fatalf("Error declarando cola monitoreo: %v", err)
	}

	msgs, err := ch.Consume("monitoreo", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error consumiendo cola: %v", err)
	}

	// Leer mensajes y emitir a suscriptores
	go func() {
		for msg := range msgs {
			var data map[string]string
			if err := json.Unmarshal(msg.Body, &data); err != nil {
				log.Printf("Error parseando JSON: %v", err)
				continue
			}
			update := &pb.EstadoEmergencia{Name: data["name"], Status: data["status"], DronId: data["dron_id"]}
			
			// Broadcast
			subsMu.Lock()
			for id, sub := range subscribers {
				if err := sub.Send(update); err != nil {
					log.Printf("Error enviando a suscriptor %d: %v", id, err)
					delete(subscribers, id)
				}
			}
			subsMu.Unlock()
		}
	}()
}

// Implementa el servicio gRPC de monitoreo
type servidorMonitoreo struct{
	pb.UnimplementedServicioMonitoreoServer
}

// RecibirActualizaciones registra un nuevo suscriptor y espera hasta que se desconecte
func (s *servidorMonitoreo) RecibirActualizaciones(_ *pb.Vacio, stream pb.ServicioMonitoreo_RecibirActualizacionesServer) error {
	subsMu.Lock()
	id := nextSubID
	nextSubID++
	subscribers[id] = stream
	subsMu.Unlock()

	log.Printf("Suscriptor %d conectado a monitoreo", id)

	// Esperar hasta que el cliente cierre
	<-stream.Context().Done()

	// Remover suscriptor
	subsMu.Lock()
	delete(subscribers, id)
	subsMu.Unlock()
	log.Printf("Suscriptor %d desconectado de monitoreo", id)

	return nil
}

func main() {
	iniciarRabbitMQ()

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
