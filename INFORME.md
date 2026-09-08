# Informe

## 1. Protocolo de comunicación

### 1.1. Forma general del mensaje

El protocolo utilizado es de tipo TLV (Type-Length-Value), con un encabezado fijo de 4 bytes:

- 2 bytes: tipo de mensaje
- 2 bytes: longitud del payload

La serialización del mismo es big-endian.

Formato:

```text
+----------------+----------------+------------------------------+
| tipo (2 bytes) | longitud (2 bytes) | payload variable         |
+----------------+----------------+------------------------------+
```

### 1.2. Tipos de mensajes

Los tipos empleados son:

- `0x01`: apuesta individual (`BET`)
- `0x02`: fin de envío de apuestas de una agencia (`END_BETS`)
- `0x03`: lista de ganadores (`WINNERS`)
- `0x04`: batch de apuestas (`NEW_BATCH`)
- `0x05`: ack (`ACK`)

En el cliente Go, estos tipos están definidos como constantes:

```go
const TLV_BET_TYPE uint16 = 0x01
const TLV_END_TYPE uint16 = 0x02
const TLV_WINNER_TYPE uint16 = 0x03
const TLV_NEW_BATCH_TYPE uint16 = 0x04
const TLV_ACK_TYPE uint16 = 0x05
```

En Python, el servidor usa constantes equivalentes.

### 1.3. Estructura de una apuesta

Cada apuesta se serializa como un bloque TLV con campos internos ordenados y etiquetados por índice. La estructura es la siguiente:

1. `agency_id`
2. `first_name`
3. `last_name`
4. `document`
5. `birthdate`
6. `number`

Cada campo se escribe como:

```text
+--------+--------+--------------------+
| índice | tamaño | valor              |
+--------+--------+--------------------+
```

Cada subcampo usa:

- 2 bytes para el identificador del campo
- 2 bytes para la longitud del valor
- N bytes para el valor en texto

### 1.4. Batch de apuestas

El cliente no envía cada apuesta individualmente. En cambio, acumula un batch de `BATCH_SIZE` apuestas y las envía bajo el tipo `NEW_BATCH`.

El flujo es:

1. lee líneas del archivo de entrada,
2. convierte cada línea a una estructura `Bet`,
3. acumula en `collected_bets`,
4. cuando `len(collected_bets) == BATCH_SIZE`, serializa el lote completo,
5. envía el mensaje de batch al servidor,
6. espera un `ACK` del servidor

La ventaja principal es reducir la sobrecarga de mensajes y mejorar la eficiencia del transporte en redes con varias agencias y más de una apuesta por agente.

### 1.5. Mensaje final de cierre del envío

Una vez que la agencia termina de procesar su archivo, envía un mensaje de tipo `END_BETS` con el `agency_id` como payload. Este mensaje indica al servidor que esa agencia ya envió todas sus apuestas.

### 1.6. Respuesta del servidor

Cuando el servidor procesa correctamente un batch, responde con `ACK`:

```text
tipo = 0x05
payload = vacio
```

Esto deja al cliente en una secuencia de protocolo bien definida:

```text
Cliente -> Servidor: batch de apuestas
Servidor -> Cliente: ACK
Cliente -> Servidor: siguiente batch
...
Cliente -> Servidor: END_BETS
Servidor -> Cliente: lista de ganadores
```

### 1.7. Lista de ganadores

Cuando el servidor valida que hay suficientes agencias y determina los ganadores para la agencia correspondiente, responde con un mensaje de tipo `WINNERS`, cuyo payload contiene una lista de apuestas ganadoras serializadas como se explico anteriormente.

## 3. Estrategias de concurrencia

### 3.1. Manejo de cada cliente en un hilo

Cada cliente es manejado en un hilo propio. Para esto, se utilizo la librería threading de python. 

### 3.2. Estado compartido y sincronización

Existen dos piezas clave para proteger la concurrencia:

1. `threading.Lock()`
   - evita que dos hilos escriban simultáneamente sobre el mismo almacenamiento.

2. `threading.Condition()`
   - sincroniza la espera de finalización de agencias,
   - permite bloquear a un hilo hasta que se cumple la condición de quorum.

La lógica es aproximadamente la siguiente:

```python
with self.condition:
    self.finished_agencies += 1
    self.condition.notify_all()
    self.condition.wait_for(
        lambda: self.finished_agencies >= self.agency_quorum_min
    )
```

Esto hace que cualquier agencia que termine su transmisión espere de forma segura hasta que el número mínimo requerido haya concluido sus envíos.

### 3.3. Elección de ganadores

La función `_choose_winners(agency_id)` recorre las apuestas guardadas y selecciona aquellas cuyo `agency_id` coincide con la agencia que termina y que cumplen la lógica de sorteo del sistema. No se hace broadcast indiscriminado de resultados; solamente se responde al cliente que solicitó el cierre con los ganadores de su agencia.


---

## 7. Conclusión

El protocolo está basado en mensajes TLV con cantidades bien definidas, que permiten serializar apuestas, batches, acknowledgments y ganadores de forma determinista. La sincronización tiene una doble finalidad: proteger el acceso compartido a las apuestas y permitir la coordinación del quorum de agencias para resolver el sorteo.

La concurrencia es central en el diseño del servidor, ya que varias agencias deben operar simultáneamente sin afectar la consistencia del sistema. Finalmente, la finalización graceful con SIGTERM y el uso de un timeout acotado son aspectos esenciales para cumplir con los requisitos de cierre ordenado y evitar fugas de recursos.

En conjunto, el diseño balancea tres objetivos clave:

- robustez en la comunicación,
- consistencia del estado compartido,
- y finalización segura y acotada del sistema.
