package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"time"

	pb "tarea_sd/proto/gen/emergencia"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

// Establece conexión con Rabbitmq para las colas que se utilizaran en el monitoreo y registro
func conectarRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial("amqp://guest:guest@10.10.28.27:5672/")
	if err != nil {
		return nil, nil, fmt.Errorf("error conectando a RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("error creando canal RabbitMQ: %v", err)
	}

	// Declarar colas
	_, err = ch.QueueDeclare("monitoreo", false, false, false, false, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error declarando cola monitoreo: %v", err)
	}

	_, err = ch.QueueDeclare("registro", false, false, false, false, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error declarando cola registro: %v", err)
	}

	return conn, ch, nil
}

// Implementa el servicio gRPC para gestión de drones
type servidorDrones struct {
	pb.UnimplementedServicioDronesServer
}

// Mapa con las posiciones iniciales de los drones
var posicionDrones = map[string][2]int32{
	"dron01": {0, 0},
	"dron02": {10, 10},
	"dron03": {-10, -10},
}

// Calcula distancia euclidiana entre dos puntos
func calcularDistancia(x1, y1, x2, y2 int32) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}

// Actualiza la posición y estado de un dron de la bd
func actualizarPosicionEnMongo(dronID string, lat, lon int32) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cliente, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://10.10.28.27:27017"))
	if err != nil {
		log.Printf("Error conectando a MongoDB: %v", err)
		return
	}
	defer cliente.Disconnect(ctx)

	coleccion := cliente.Database("emergencias").Collection("drones")

	filtro := bson.M{"id": dronID}
	actualizacion := bson.M{
		"$set": bson.M{
			"latitude":  lat,
			"longitude": lon,
			"status":    "available",
		},
	}

	_, err = coleccion.UpdateOne(ctx, filtro, actualizacion)
	if err != nil {
		log.Printf("Error actualizando dron en MongoDB: %v", err)
		return
	}

	log.Printf("Dron %s actualizado en MongoDB a (%d, %d)", dronID, lat, lon)
}

// Implementación del servicio gRPC para asignación de emergencias a drones
func (s *servidorDrones) AsignarEmergencia(ctx context.Context, asig *pb.Asignacion) (*pb.Vacio, error) {
	emergencia := asig.EmergencyName
	dron := asig.DronId

	pos := posicionDrones[dron]
	fmt.Printf(" %s asignado a emergencia '%s'\n", dron, emergencia)
	// Conectarse a MongoDB para obtener coordenadas y magnitud de la emergencia
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clienteMongo, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://10.10.28.27:27017"))
	if err != nil {
		log.Printf("Error conectando a MongoDB: %v", err)
		return &pb.Vacio{}, nil
	}
	defer clienteMongo.Disconnect(ctx)

	coleccion := clienteMongo.Database("emergencias").Collection("emergencias")

	var doc struct {
		Name      string `bson:"name"`
		Latitude  int32  `bson:"latitude"`
		Longitude int32  `bson:"longitude"`
		Magnitude int32  `bson:"magnitude"`
	}

	err = coleccion.FindOne(ctx, bson.M{"name": emergencia}).Decode(&doc)
	if err != nil {
		log.Printf("No se encontró la emergencia '%s' en MongoDB: %v", emergencia, err)
		return &pb.Vacio{}, nil
	}

	// Calcula ruta y tiempos reales
	dest := [2]int32{doc.Latitude, doc.Longitude}
	distancia := calcularDistancia(pos[0], pos[1], dest[0], dest[1])
	tiempoViaje := time.Duration(distancia * 0.5 * float64(time.Second))
	duracionApagado := time.Duration(doc.Magnitude) * 2 * time.Second

	fmt.Printf("%s viajando a emergencia '%s' en (%d,%d), distancia %.2f, magnitud %d\n", dron, emergencia, dest[0], dest[1], distancia, doc.Magnitude)

	// Conexión a RabbitMQ
	conn, canal, err := conectarRabbitMQ()
	if err != nil {
		log.Printf("Error en RabbitMQ: %v", err)
		return &pb.Vacio{}, nil
	}
	defer conn.Close()
	defer canal.Close()

	// 1. Publicar "en camino" cada 5s durante el viaje
	for t := time.Duration(0); t < tiempoViaje; t += 5 * time.Second {
		msg := map[string]string{
			"name":    emergencia,
			"status":  "Dron en camino a emergencia...",
			"dron_id": dron,
		}
		body, _ := json.Marshal(msg)
		canal.Publish("", "monitoreo", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
		log.Printf("Enviado a monitoreo: %s", body)
		time.Sleep(5 * time.Second)
	}

	// 2. Publicar "apagando" cada 5s durante el apagado
	for t := time.Duration(0); t < duracionApagado; t += 5 * time.Second {
		msg := map[string]string{
			"name":    emergencia,
			"status":  "Dron apagando emergencia...",
			"dron_id": dron,
		}
		body, _ := json.Marshal(msg)
		canal.Publish("", "monitoreo", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
		log.Printf("Enviado a monitoreo: %s", body)
		time.Sleep(5 * time.Second)
	}

	// 3. Al finalizar, publicar mensaje final de extinción
	msgFinal := map[string]string{
		"name":    emergencia,
		"status":  fmt.Sprintf("%s ha sido extinguido por %s", emergencia, dron),
		"dron_id": dron,
	}
	bodyFinal, _ := json.Marshal(msgFinal)
	canal.Publish("", "monitoreo", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        bodyFinal,
	})
	log.Printf("Emergencia extinguida: %s", bodyFinal)

	//Actualización de posición final del dron
	posicionDrones[dron] = dest
	actualizarPosicionEnMongo(dron, dest[0], dest[1])
	fmt.Printf("%s apagó emergencia '%s' y quedó en posición (%d,%d)\n", dron, emergencia, dest[0], dest[1])

	// Enviar a cola "registro" con status Extinguido
	finalMsg := map[string]string{
		"name":    emergencia,
		"status":  "Extinguido",
		"dron_id": dron,
	}
	body, _ := json.Marshal(finalMsg)
	canal.Publish("", "registro", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
	log.Printf("Enviado a registro: %s", body)

	return &pb.Vacio{}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("No se pudo iniciar servidor de drones: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterServicioDronesServer(grpcServer, &servidorDrones{})

	fmt.Println("Servicio de drones escuchando en puerto 50053")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error al servir: %v", err)
	}
}
