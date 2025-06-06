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

## Proceso de ejecución
### MV 1
- go run cliente/cliente.go emergencia.json
- go run monitoreo/monitoreo.go
### MV 2
- go run asignacion/asignacion.go
- python3 registro/registro.py
### MV 3
- go run drones/drones.go
