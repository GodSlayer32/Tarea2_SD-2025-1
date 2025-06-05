import pika
import pymongo
import json

#Se hace conexión a MongoDB
cliente_mongo = pymongo.MongoClient("mongodb://localhost:27017/")
db = cliente_mongo["emergencias"]
coleccion = db["emergencias"]

#Conexión a RabbitMQ para el server de mensajería
conexion = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
canal = conexion.channel()
canal.queue_declare(queue='registro')

print("Esperando mensajes en la cola 'registro'...")

    
    #Función que procesa los mensajes recibidos de RabbitMQ.
    #Maneja dos tipos de estados: 'En curso' y 'Extinguido' 
    
def callback(ch, method, properties, body):
    data = json.loads(body)
    nombre = data.get("name")
    status = data.get("status")
    print(f"Mensaje recibido: {nombre} — Estado: {status}")

    if status == "En curso":
        #Verifica si la emergencia ya existe en la base de datos
        existente = coleccion.find_one({"name": nombre})
        if existente is None:
            emergencia = {
                "name": nombre,
                "latitude": data.get("latitude", 0),
                "longitude": data.get("longitude", 0),
                "magnitude": data.get("magnitude", 0),
                "status": status
            }
            
            #Si no lo agrega
            coleccion.insert_one(emergencia)
            print(f"Emergencia '{nombre}' insertada en MongoDB.")
        else:
            print(f"Emergencia '{nombre}' ya existe. No se duplica.")

    elif status == "Extinguido":
        # Actualizar solo si existe y no está ya extinguida
        result = coleccion.update_one(
            {"name": nombre, "status": {"$ne": "Extinguido"}},
            {"$set": {"status": "Extinguido"}}
        )
        if result.modified_count > 0:
            print(f"Emergencia '{nombre}' actualizada a 'Extinguido'.")
        else:
            print(f"Emergencia '{nombre}' ya estaba marcada como 'Extinguido' o no existe.")


canal.basic_consume(queue='registro', on_message_callback=callback, auto_ack=True)

try:
    canal.start_consuming()
except KeyboardInterrupt:
    print("Interrumpido por el usuario")
    canal.stop_consuming()
    conexion.close()
