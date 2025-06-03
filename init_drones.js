use emergencias;

db.drones.deleteMany({});  // Limpiar si ya hay drones

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
