import pika
import pymongo
import json
import pprint

# Conexión a MongoDB
cliente_mongo = pymongo.MongoClient("mongodb://10.10.28.27:27017/")
db = cliente_mongo["emergencias"]
coleccion = db["emergencias"]

# Conexión a RabbitMQ
conexion = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
canal = conexion.channel()
canal.queue_declare(queue='registro')

print("Esperando mensajes en la cola 'registro'...")

# Función para exportar la colección de emergencias a archivo
def exportar_emergencias_a_archivo():
    datos = list(coleccion.find({}))
    with open("./database.mongo", "w", encoding="utf-8") as f:
        for doc in datos:
            doc.pop("_id", None)  # eliminar el campo _id
            f.write(pprint.pformat(doc, indent=2))
            f.write(",\n")

# Función para procesar mensajes de RabbitMQ
def callback(ch, method, properties, body):
    data = json.loads(body)
    nombre = data.get("name")
    status = data.get("status")
    print(f"Mensaje recibido: {nombre} — Estado: {status}")

    if status == "En curso":
        existente = coleccion.find_one({"name": nombre})
        if existente is None:
            emergencia = {
                "name": nombre,
                "latitude": data.get("latitude", 0),
                "longitude": data.get("longitude", 0),
                "magnitude": data.get("magnitude", 0),
                "status": status
            }
            coleccion.insert_one(emergencia)
            print(f"Emergencia '{nombre}' insertada en MongoDB.")
            exportar_emergencias_a_archivo()
        else:
            print(f"Emergencia '{nombre}' ya existe. No se duplica.")

    elif status == "Extinguido":
        result = coleccion.update_one(
            {"name": nombre, "status": {"$ne": "Extinguido"}},
            {"$set": {"status": "Extinguido"}}
        )
        if result.modified_count > 0:
            print(f"Emergencia '{nombre}' actualizada a 'Extinguido'.")
            exportar_emergencias_a_archivo()
        else:
            print(f"Emergencia '{nombre}' ya estaba marcada como 'Extinguido' o no existe.")

canal.basic_consume(queue='registro', on_message_callback=callback, auto_ack=True)

try:
    canal.start_consuming()
except KeyboardInterrupt:
    print("Interrumpido por el usuario")
    canal.stop_consuming()
    conexion.close()
