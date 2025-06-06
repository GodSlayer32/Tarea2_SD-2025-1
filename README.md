# Tarea 2 — INF-343 Sistemas Distribuidos
## Sistema distribuido de gestión de emergencias con drones

### Integrantes del grupo
| Nombre                | Rol USM       |
|-----------------------|---------------|
| Catalina Yañez        | 202273010-2   |
| Javier Miranda        | 202104073-0   |
| Eduardo Pacheco       | 202273014-5   |

---

### Estructura de componentes

| Componente     | Lenguaje | Archivo                   | Máquina Virtual |
|----------------|----------|---------------------------|------------------|
| Cliente        | Go       | `cliente/cliente.go`      | MV1              |
| Monitoreo      | Go       | `monitoreo/monitoreo.go`  | MV1              |
| Asignación     | Go       | `asignacion/asignacion.go`| MV2              |
| Registro       | Python   | `registro/registro.py`    | MV2              |
| MongoDB        | -        | `init_drones.js`          | MV2              |
| Drones         | Go       | `drones/drones.go`        | MV3              |

---

### Instrucciones de ejecución

#### Requisitos

- RabbitMQ en todas las máquinas:
  ```bash
  sudo systemctl start rabbitmq-server

## Distribución MVs
### MV 1
1. go run cliente/cliente.go emergencia.json
2. go run monitoreo/monitoreo.go
### MV 2
3. go run asignacion/asignacion.go
4. python3 registro/registro.py
### MV 3
5. go run drones/drones.go

## Proceso de ejecución 
1. Primero en la MV3 ejecutar el script para poblar la bd de los drones: mongosh < database/init_drones.js 
2. Luego según la enumeración en la distribución de Mvs, sería 3,4,5,2,1