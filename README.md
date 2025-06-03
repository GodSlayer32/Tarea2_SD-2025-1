# Tarea 2 — INF-343 Sistemas Distribuidos
## Sistema distribuido de gestión de emergencias con drones

### 👥 Integrantes del grupo
| Nombre                | Rol USM       |
|-----------------------|---------------|
| Nombre Apellido       | 202573844-X   |
| Nombre Apellido       | 202573821-3   |
| Nombre Apellido       | 202573888-9   |

---

### 🗂 Estructura de componentes

| Componente     | Lenguaje | Archivo               | Máquina Virtual |
|----------------|----------|------------------------|------------------|
| Cliente        | Go       | `cliente/cliente.go`   | MV1              |
| Monitoreo      | Go       | `monitoreo.go`         | MV1              |
| Asignación     | Go       | `asignacion.go`        | MV2              |
| Registro       | Python   | `registro.py`          | MV2              |
| MongoDB        | -        | `init_drones.js`       | MV2              |
| Drones         | Go       | `drones.go`            | MV3              |

---

### 🧪 Instrucciones de ejecución

#### Requisitos

- RabbitMQ en todas las máquinas:
  ```bash
  sudo systemctl start rabbitmq-server
