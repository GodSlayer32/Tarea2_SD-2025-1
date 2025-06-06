package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	pb "tarea_sd/proto/gen/emergencia"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// Canal para compartir mensajes desde RabbitMQ al gRPC (buffer 100)
var updatesChan = make(chan *pb.EstadoEmergencia, 100)

// Conexión a RabbitMQ y escucha de la cola "monitoreo"
func iniciarRabbitMQ() {
	conn, err := amqp.Dial("amqp://guest:guest@10.10.28.27:5672/")
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}
	// Crea un canal de comunicación
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creando canal: %v", err)
	}

	// Declara la cola "monitoreo" si no existe
	_, err = ch.QueueDeclare("monitoreo", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error declarando cola monitoreo: %v", err)
	}

	msgs, err := ch.Consume(
		"monitoreo", "", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Error consumiendo cola: %v", err)
	}

	// Procesar mensajes
	go func() {
		for msg := range msgs {
			var data map[string]string
			// Decodifica el mensaje JSON
			if err := json.Unmarshal(msg.Body, &data); err != nil {
				log.Printf("Error parseando JSON: %v", err)
				continue
			}

			// Crea estructura de estado de emergencia
			update := &pb.EstadoEmergencia{
				Name:   data["name"],
				Status: data["status"],
				DronId: data["dron_id"],
			}

			log.Printf("Recibido de RabbitMQ: %v", update)
			updatesChan <- update
		}
	}()
}

// Implementa el servicio gRPC de monitoreo
type servidorMonitoreo struct {
	pb.UnimplementedServicioMonitoreoServer
}

func (s *servidorMonitoreo) RecibirActualizaciones(_ *pb.Vacio, stream pb.ServicioMonitoreo_RecibirActualizacionesServer) error {
	log.Println("Cliente conectado a monitoreo")

	// Creamos canal local por cliente
	clienteChan := make(chan *pb.EstadoEmergencia, 100)

	// Iniciamos nueva goroutine que reenvía desde el canal global solo cuando el cliente está conectado
	go func() {
		for update := range updatesChan {
			clienteChan <- update
		}
	}()

	// Ahora enviamos al cliente desde clienteChan
	for update := range clienteChan {
		if err := stream.Send(update); err != nil {
			log.Printf("Error enviando al cliente: %v", err)
			break
		}
	}

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
