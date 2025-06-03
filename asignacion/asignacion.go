package main

import (
    "context"
    "fmt"
    "log"
    "math"
    "net"
    "time"

    "google.golang.org/grpc"
    pb "tarea_sd/proto/gen/emergencia"
    "encoding/json"
    amqp "github.com/rabbitmq/amqp091-go"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type servidorAsignacion struct {
    pb.UnimplementedServicioAsignacionServer
}

func calcularDistancia(x1, y1, x2, y2 float64) float64 {
    dx := x2 - x1
    dy := y2 - y1
    return math.Sqrt(dx*dx + dy*dy)
}

func obtenerDronDisponible(lat, lon int32) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cliente, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        return "", err
    }
    defer cliente.Disconnect(ctx)

    coleccion := cliente.Database("emergencias").Collection("drones")

    cursor, err := coleccion.Find(ctx, bson.M{"status": "available"})
    if err != nil {
        return "", err
    }
    defer cursor.Close(ctx)

    var mejorDron string
    minDist := math.MaxFloat64

    for cursor.Next(ctx) {
        var dron struct {
            ID        string  `bson:"id"`
            Latitude  float64 `bson:"latitude"`
            Longitude float64 `bson:"longitude"`
        }

        if err := cursor.Decode(&dron); err != nil {
            continue
        }

        dist := calcularDistancia(float64(lat), float64(lon), dron.Latitude, dron.Longitude)
        if dist < minDist {
            minDist = dist
            mejorDron = dron.ID
        }
    }

    if mejorDron == "" {
        return "", fmt.Errorf("no hay drones disponibles")
    }

    // Marcar el dron como ocupado
    _, err = coleccion.UpdateOne(ctx, bson.M{"id": mejorDron}, bson.M{
        "$set": bson.M{"status": "busy"},
    })
    if err != nil {
        return "", err
    }

    return mejorDron, nil
}
func publicarEnRegistro(e *pb.Emergencia) {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Printf("❌ Error conectando a RabbitMQ: %v", err)
        return
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Printf("❌ Error abriendo canal RabbitMQ: %v", err)
        return
    }
    defer ch.Close()

    ch.QueueDeclare("registro", false, false, false, false, nil)

    msg := map[string]interface{}{
        "name":      e.Name,
        "latitude":  e.Latitude,
        "longitude": e.Longitude,
        "magnitude": e.Magnitude,
        "status":    "En curso",
    }

    body, _ := json.Marshal(msg)

    ch.Publish("", "registro", false, false, amqp.Publishing{
        ContentType: "application/json",
        Body:        body,
    })

    log.Printf("📝 Publicado en registro (En curso): %s", body)
}


func (s *servidorAsignacion) EnviarEmergencias(stream pb.ServicioAsignacion_EnviarEmergenciasServer) error {
    fmt.Println("📨 Recibiendo emergencias del cliente...")

    for {
        emergencia, err := stream.Recv()
        if err != nil {
            fmt.Println("✅ Fin de stream de emergencias.")
            break
        }

        fmt.Printf("🚨 Emergencia recibida: %s (%d,%d)\n", emergencia.Name, emergencia.Latitude, emergencia.Longitude)

        // Elegir el dron más cercano
        dronID, err := obtenerDronDisponible(emergencia.Latitude, emergencia.Longitude)
        if err != nil {
            log.Printf("❌ No se pudo asignar dron: %v", err)
            continue
        }

        fmt.Printf("✅ Dron asignado: %s\n", dronID)
		publicarEnRegistro(emergencia)


        // Enviar asignación al servicio de drones
        conn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
        if err != nil {
            log.Printf("❌ No se pudo conectar al servicio de drones: %v", err)
            continue
        }
        defer conn.Close()

        clienteDrones := pb.NewServicioDronesClient(conn)
        _, err = clienteDrones.AsignarEmergencia(context.Background(), &pb.Asignacion{
            EmergencyName: emergencia.Name,
            DronId:        dronID,
        })
        if err != nil {
            log.Printf("❌ Error enviando asignación a drones: %v", err)
        }
    }

    return stream.SendAndClose(&pb.Vacio{})
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("❌ No se pudo iniciar servidor de asignación: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterServicioAsignacionServer(grpcServer, &servidorAsignacion{})

    fmt.Println("🚦 Servicio de asignación escuchando en puerto 50051")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("❌ Error al servir: %v", err)
    }
}
