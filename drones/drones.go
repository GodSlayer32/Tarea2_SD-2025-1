package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "google.golang.org/grpc"
    pb "tarea_sd/proto/gen/emergencia"
)

func conectarRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
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

type servidorDrones struct {
    pb.UnimplementedServicioDronesServer
}

var posicionDrones = map[string][2]int32{
    "dron01": {0, 0},
    "dron02": {10, 10},
    "dron03": {-10, -10},
}

func calcularDistancia(x1, y1, x2, y2 int32) float64 {
    dx := float64(x2 - x1)
    dy := float64(y2 - y1)
    return math.Sqrt(dx*dx + dy*dy)
}

func actualizarPosicionEnMongo(dronID string, lat, lon int32) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cliente, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Printf("❌ Error conectando a MongoDB: %v", err)
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
        log.Printf("❌ Error actualizando dron en MongoDB: %v", err)
        return
    }

    log.Printf("📍 Dron %s actualizado en MongoDB a (%d, %d)", dronID, lat, lon)
}

func (s *servidorDrones) AsignarEmergencia(ctx context.Context, asig *pb.Asignacion) (*pb.Vacio, error) {
    emergencia := asig.EmergencyName
    dron := asig.DronId

    pos := posicionDrones[dron]
    fmt.Printf("🚁 %s asignado a emergencia '%s'\n", dron, emergencia)

    coordsEmergencia := map[string][2]int32{
        "Incendio Forestal Sur": {50, 10},
        "Incendio de palets":    {-25, 30},
        "Incendio de bus":       {25, 25},
    }

    dest := coordsEmergencia[emergencia]
    distancia := calcularDistancia(pos[0], pos[1], dest[0], dest[1])
    tiempoViaje := time.Duration(distancia * 0.5 * float64(time.Second))
    fmt.Printf("🧭 %s viajando a emergencia (%d,%d), distancia %.2f\n", dron, dest[0], dest[1], distancia)

    // Conexión a RabbitMQ
    conn, canal, err := conectarRabbitMQ()
    if err != nil {
        log.Printf("❌ Error en RabbitMQ: %v", err)
        return &pb.Vacio{}, nil
    }
    defer conn.Close()
    defer canal.Close()

    ticker := time.NewTicker(5 * time.Second)
    done := make(chan bool)

    go func() {
        for {
            select {
            case <-ticker.C:
                msg := map[string]string{
                    "name":    emergencia,
                    "status":  "En curso",
                    "dron_id": dron,
                }
                body, _ := json.Marshal(msg)
                canal.Publish("", "monitoreo", false, false, amqp.Publishing{
                    ContentType: "application/json",
                    Body:        body,
                })
                log.Printf("📡 Enviado a monitoreo: %s", body)
            case <-done:
                return
            }
        }
    }()

    time.Sleep(tiempoViaje)
    ticker.Stop()
    done <- true

    fmt.Printf("🔥 %s apagando emergencia '%s'\n", dron, emergencia)
    magnitudes := map[string]int32{
        "Incendio Forestal Sur": 5,
        "Incendio de palets":    1,
        "Incendio de bus":       3,
    }
    duracionApagado := time.Duration(magnitudes[emergencia]) * 2 * time.Second
    time.Sleep(duracionApagado)

    posicionDrones[dron] = dest
    actualizarPosicionEnMongo(dron, dest[0], dest[1])
    fmt.Printf("✅ %s apagó emergencia '%s' y quedó en posición (%d,%d)\n", dron, emergencia, dest[0], dest[1])

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
    log.Printf("📝 Enviado a registro: %s", body)

    return &pb.Vacio{}, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50053")
    if err != nil {
        log.Fatalf("❌ No se pudo iniciar servidor de drones: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterServicioDronesServer(grpcServer, &servidorDrones{})

    fmt.Println("🚁 Servicio de drones escuchando en puerto 50053")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("❌ Error al servir: %v", err)
    }
}
