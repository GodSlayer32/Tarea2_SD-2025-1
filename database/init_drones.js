use emergencias;

db.emergencias.deleteMany({});
db.drones.deleteMany({});  // Eliminar drones anteriores

db.drones.insertMany([
  {
    id: "dron01",
    latitude: -50,
    longitude: -50,
    status: "available"
  },
  {
    id: "dron02",
    latitude: 50,
    longitude: -50,
    status: "available"
  },
  {
    id: "dron03",
    latitude: 0,
    longitude: 50,
    status: "available"
  }
]);

print("✅ Drones insertados exitosamente");
