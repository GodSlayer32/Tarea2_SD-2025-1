// Conexión a RabbitMQ y escucha de la cola "monitoreo"
func iniciarRabbitMQ() {
	// Establece la conexión con el servidor de RabbitMQ usando las credenciales por defecto
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}

	// Crea un canal de comunicación sobre la conexión establecida
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creando canal: %v", err)
	}

	// Declara la cola "monitoreo". Si no existe, la crea.
	_, err = ch.QueueDeclare("monitoreo", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error declarando cola monitoreo: %v", err)
	}

	// Se suscribe a la cola "monitoreo" para comenzar a consumir mensajes.
	// Los mensajes se entregan automáticamente (auto-ack: true)
	msgs, err := ch.Consume(
		"monitoreo", // nombre de la cola
		"",          // consumer tag
		true,        // auto acknowledge (reconocimiento automático)
		false,       // no exclusivo
		false,       // no esperar al servidor
		false,       // sin argumentos adicionales
		nil,
	)
	if err != nil {
		log.Fatalf("Error consumiendo cola: %v", err)
	}

	// Inicia para procesar los mensajes recibidos
	go func() {
		for msg := range msgs {
			var data map[string]string

			// Intenta parsear el cuerpo del mensaje (JSON) a un mapa clave-valor
			if err := json.Unmarshal(msg.Body, &data); err != nil {
				log.Printf("Error parseando JSON: %v", err)
				continue
			}

			// Construye la estructura EstadoEmergencia usando los datos del mensaje
			update := &pb.EstadoEmergencia{
				Name:   data["name"],
				Status: data["status"],
				DronId: data["dronid"],
			}

			// Muestra por consola el mensaje recibido
			log.Printf("Recibido de RabbitMQ: %v", update)

			// Envía la actualización al canal para que sea recogida por el servidor gRPC
			updatesChan <- update
		}
	}()
}
