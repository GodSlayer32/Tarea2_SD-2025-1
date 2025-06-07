# Tarea 2 — INF-343 Sistemas Distribuidos

## Sistema distribuido de gestión de emergencias con drones

### Integrantes del grupo

| Nombre          | Rol USM     |
| --------------- | ----------- |
| Catalina Yañez  | 202273010-2 |
| Javier Miranda  | 202104073-0 |
| Eduardo Pacheco | 202273014-5 |

---

### Estructura de componentes

| Componente | Lenguaje | Archivo                    | Máquina Virtual   |
| ---------- | -------- | -------------------------- | ----------------- |
| Cliente    | Go       | `cliente/cliente.go`       | MV1 (10.10.28.26) |
| Monitoreo  | Go       | `monitoreo/monitoreo.go`   | MV1 (10.10.28.26) |
| Asignación | Go       | `asignacion/asignacion.go` | MV2 (10.10.28.27) |
| Registro   | Python   | `registro/registro.py`     | MV2 (10.10.28.27) |
| MongoDB    | -        | `database/init_drones.js`  | MV2 (10.10.28.27) |
| Drones     | Go       | `drones/drones.go`         | MV3 (10.10.28.28) |

---

### Proceso de ejecución

A continuación, el paso a paso desde abrir las terminales hasta levantar todos los servicios.

#### 1. Abrir terminales SSH en cada VM

* **MV1** (Cliente + Monitoreo):

  ```bash
  ssh ubuntu@10.10.28.26
  cd ~/Tarea2_SD-2025-1
  ```

* **MV2** (Asignación + Registro + MongoDB):

  ```bash
  ssh ubuntu@10.10.28.27
  cd ~/Tarea2_SD-2025-1
  ```

* **MV3** (Drones):

  ```bash
  ssh ubuntu@10.10.28.28
  cd ~/Tarea2_SD-2025-1
  ```

#### 2. Preparar entorno en MV2 (Registro)

En la terminal de **MV2**:

1. Crear y activar el entorno virtual de Python:

   ```bash
   python3 -m venv venv
   source venv/bin/activate
   ```
2. Instalar dependencias:

   ```bash
   pip install -r requirements.txt
   ```
3. Poblar la colección de drones en MongoDB:

   ```bash
   mongosh < database/init_drones.js
   ```

#### 3. Levantar servicios en orden

##### En MV2:

1. Iniciar RabbitMQ (si no está corriendo):

   ```bash
   sudo systemctl start rabbitmq-server
   ```
2. Iniciar MongoDB:

   ```bash
   sudo systemctl start mongod
   ```

##### En MV3:

3. Ejecutar Drones:

   ```bash
   go run drones/drones.go
   ```

##### En MV2:

4. Ejecutar Asignación:

   ```bash
   go run asignacion/asignacion.go
   ```
5. Ejecutar Registro (en el mismo `venv`):

   ```bash
   source venv/bin/activate   # si no está activo
   python3 registro/registro.py
   ```

##### En MV1:

6. Ejecutar Monitoreo:

   ```bash
   go run monitoreo/monitoreo.go
   ```
7. Finalmente, ejecutar Cliente:

   ```bash
   go run cliente/cliente.go emergencia.json
   ```

---

Con este orden y configuración, el sistema completo quedará levantado y podrá simular la gestión de emergencias con drones.
