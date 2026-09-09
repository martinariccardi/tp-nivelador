# Informe

## 1. Arquitectura general

El sistema tiene una arquitectura cliente-servidor:

- **Cliente** (Go): representa a una agencia de apuestas. Lee un archivo CSV de apuestas, las agrupa en *batches* y las envía al servidor. Al finalizar, recibe la lista de ganadores de su agencia y la escribe en un archivo de salida.
- **Servidor** (Python): acepta conexiones de múltiples agencias en simultáneo, cada una atendida en un hilo propio (`threading.Thread`). Almacena las apuestas recibidas y, cuando se alcanza el quorum mínimo de agencias, calcula y envía los ganadores correspondientes a cada una.

La comunicación entre ambos servicios está definida por un protocolo basado en TLV.

## 2. Protocolo de comunicación

Como se menciono en la sección anterior, se implementó un protocolo de comunicación basado en TLV, el cual sera detallado en las siguientes secciones.

### 2.1. Forma general del mensaje

El protocolo utilizado es de tipo TLV (Type-Length-Value), con un encabezado fijo de 4 bytes:

- 2 bytes: tipo de mensaje
- 2 bytes: longitud del payload

La serialización es big-endian.

Formato:

```text
+----------------+---------------------+---------------------------+
| tipo (2 bytes) | longitud (2 bytes)  | payload variable         |
+----------------+---------------------+---------------------------+
```

### 2.2. Tipos de mensajes

Los tipos empleados son:

| Valor  | Nombre             | Descripción                                   |
|--------|--------------------|-----------------------------------------------|
| `0x01` | `BET`              | Apuesta individual                            |
| `0x02` | `END_BETS`         | Fin de envío de apuestas de una agencia       |
| `0x03` | `WINNERS`          | Lista de ganadores                            |
| `0x04` | `NEW_BATCH`        | Batch de apuestas                             |
| `0x05` | `ACK`              | Confirmación de recepción (todo ok)           |
| `0x06` | `NACK`             | Confirmación de recepción con error           |

En el cliente Go, estos tipos están definidos como constantes:

```go
const TLV_BET_TYPE uint16 = 0x01
const TLV_END_TYPE uint16 = 0x02
const TLV_WINNER_TYPE uint16 = 0x03
const TLV_NEW_BATCH_TYPE uint16 = 0x04
const TLV_ACK_TYPE uint16 = 0x05
const TLV_NACK_TYPE uint16 = 0x06
```

En Python, el servidor (`protocol.py`) usa constantes equivalentes. 

### 2.3. Estructura de una apuesta

Cada apuesta se serializa como un bloque TLV compuesto por 6 subcampos ordenados y etiquetados por índice:

1. `agency_id`
2. `first_name`
3. `last_name`
4. `document`
5. `birthdate`
6. `number`

Cada subcampo se escribe como:

```text
+-------------------+-------------------+--------------------+
| tipo (2 bytes)  | longitud (2 bytes)| valor (N bytes)    |
+-------------------+-------------------+--------------------+
```

#### Ejemplo de una apuesta: 

Supongamos la siguiente apuesta:

agency_id = "1"
first_name = "Juan"
last_name = "Perez"
document = "12345678"
birthdate = "1999-03-17"
number = "7574"

Cada subcampo se codifica como índice (2 bytes) + longitud (2 bytes) + valor, y luego todo el conjunto se envuelve en el TLV externo de tipo BET (0x01):

Serialización
00 01 00 36 00 01 00 01 31 00 02 00 03 41 6e 61
00 03 00 04 44 69 61 7a 00 04 00 08 33 30 39 30
34 34 36 35 00 05 00 0a 31 39 39 39 2d 30 33 2d
31 37 00 06 00 04 37 35 37 34

Serialización (desglose)
00 01 00 36                              (TLV externo: tipo=BET, longitud=54)
     00 01 00 01 31                      (índice 1, longitud 1: agency_id = "1")
     00 02 00 03 41 6e 61                (índice 2, longitud 3: first_name = "Ana")
     00 03 00 04 44 69 61 7a             (índice 3, longitud 4: last_name = "Diaz")
     00 04 00 08 33 30 39 30 34 34 36 35 (índice 4, longitud 8: document = "30904465")
     00 05 00 0a 31 39 39 39 2d 30 33 2d
     31 37                               (índice 5, longitud 10: birthdate = "1999-03-17")
     00 06 00 04 37 35 37 34             (índice 6, longitud 4: number = "7574")

Todos los subcampos se serializan como texto (UTF-8), incluso los que representan números. No hay campos binarios de tamaño fijo como en un protocolo tipo integer/string tipado, sino que todo viaja como string con su longitud explícita.

### 2.4. Batching

El cliente no envía cada apuesta individualmente. Sino que, acumula un lote de `BatchSize` apuestas y las envía bajo el tipo `NEW_BATCH`.

El flujo es:

1. lee líneas del archivo de entrada,
2. convierte cada línea a una estructura `Bet` (agregando el `agency_id` de configuración),
3. guarda las apuestas en `collectedBets`,
4. cuando `len(collectedBets) == BatchSize`, serializa el lote completo,
5. envía el mensaje de batch al servidor,
6. espera un `ACK` (o `NACK`) del servidor (si no recibe un `ACK`, no puede mandar el siguiente batch).

Si al terminar de leer el archivo quedan apuestas acumuladas que no llegaron a completar un batch, el cliente las envía igual como un batch final más pequeño, antes de notificar el fin de la transmisión. Esto garantiza que ninguna apuesta se pierda. 

### 2.5. Mensaje final de cierre del envío

Una vez que la agencia termina de procesar su archivo (incluyendo el envío del batch final), envía un mensaje de tipo `END_BETS` con el `agency_id` como payload. Este mensaje indica al servidor que esa agencia ya envió todas sus apuestas.

### 2.6. Respuesta del servidor a un batch

Cuando el servidor procesa correctamente un batch, responde con `ACK`:

```text
tipo = 0x05
payload = vacío
```

Si el batch recibido está mal formado (no puede deserializarse), el servidor está pensado para responder con `NACK` (`tipo = 0x06`) en lugar de cortar la conexión.

De esta forma, el flujo de mensajes entre el cliente y el servidor es el siguiente:

```text
Cliente -> Servidor: batch de apuestas
Servidor -> Cliente: ACK / NACK
Cliente -> Servidor: siguiente batch
...
Cliente -> Servidor: END_BETS
Servidor -> Cliente: lista de ganadores
```

### 2.7. Lista de ganadores

Cuando el servidor valida que hay suficientes agencias y determina los ganadores para la agencia correspondiente, responde con un mensaje de tipo `WINNERS`, el cual contiene una lista de apuestas ganadoras serializadas con el mismo formato descripto en 2.3.

El cliente, al recibir este mensaje, escribe cada ganador en el archivo de salida con el formato `first_name,last_name,document,birthdate,number`.

## 3. Estrategias de concurrencia

### 3.1. Manejo de cada cliente en un hilo

Cada conexión aceptada por el servidor es manejada en un hilo propio (`threading.Thread`), lo que permite atender a varias agencias en simultáneo sin bloquear el `accept()` del socket principal.

### 3.2. Estado compartido y sincronización

Para manejar la concurrencia se utilizaron 2 herramientas:

1. `threading.Lock()`
   - evita que dos hilos escriban simultáneamente sobre el mismo archivo de apuestas.

2. `threading.Condition()`
   - sincroniza la espera de finalización de agencias,
   - permite bloquear a un hilo hasta que se cumple la condición de quorum.

La lógica es aproximadamente la siguiente:

```python
with self.condition:
    self.finished_agencies += 1
    self.condition.notify_all()
    self.condition.wait_for(
        lambda: self.finished_agencies >= self.agency_quorum_min or not self.running
    )
```

Esto hace que cualquier agencia que termine su transmisión espere (sin busy-wait) hasta que el número mínimo requerido de agencias haya terminado. La condición adicional `or not self.running` evita que un hilo quede bloqueado indefinidamente si el servidor esta terminando si ejecución.

### 3.3. Elección de ganadores

La función `_choose_winners(agency_id)` recorre las apuestas guardadas y solo selecciona aquellas cuyo `agency_id` coincide con la agencia que solicita el cierre y que, además, hayan ganado (`self.lottery.has_won(bet)`). No se hace *broadcast* de resultados; solamente se responde al cliente que solicitó el cierre con los ganadores de su propia agencia, tal como pedía la consigna.

# 3.4. Por qué usar threading no es un problema pese al GIL

Como se menciono en las secciones anteriores para el manejo de la concurrencia se opto por utilizar la librería de Python `Threading`. Esta librería permite ejecutar tareas concurrente a traves del uso de hilos o threads. 

Python tiene un mecanismo interno llamado GIL (Global Interpreter Lock). Resumidamente, es un lock único que hace que, en un momento dado, solo un hilo pueda estar ejecutando código Python, aunque hayamos creado varios hilos. Esto existe porque el manejador de memoria interno de Python no está preparado para que dos hilos lo toquen al mismo tiempo de forma segura. 

Con esto pareciera que el uso de threads en python no es muy util, mas que nada en tareas con un uso intensivo de la CPU. Sin embargo, en este caso,el servidor implementado no realiza tareas con un uso intensivo de la CPU, sino que se pasa la mayoría del tiempo esperando cosas. Como por ejemplo, espera que le llegue una nueva conexión, espera que le llegue un nuevo batch, etc. De esta forma, cuando un hilo se pone a esperar algo de la red, Python suelta el lock. No lo necesita porque no está ejecutando nada, solo está esperando.

Entonces, mientras el hilo de la Agencia 1 está esperando que le lleguen más datos por el socket, el lock se libera y el hilo de la Agencia 2 lo puede agarrar y seguir trabajando (parsear su batch, guardar las apuestas, etc.). Como se "espera" la mayor parte del tiempo de la vida de cada hilo, en la práctica sí se logra que varias agencias avancen concurrentemente, incluso con el lock. 

Es por eso, que para este caso particular usar la librería threading es valido y cumple con la consigna.

## 5. Finalización graceful (SIGTERM)

Para asegurarnos de que el programa termine de forma limpia y ordenada, se maneja la llegada de la señal `SIGTERM` tanto en el cliente como en el servidor. 

En el caso del servidor, al recibir la señal de terminación del proceso se ejecuta una función que realiza un apagado ordenado con el siguiente flujo:

1. Se marca `self.running = False`, lo que hace que el loop principal de `accept()` y los loops de manejo de cada cliente dejen de aceptar nuevas conexiones.
2. Se notifica a todos los hilos que puedan estar bloqueados esperando el quorum (`self.condition.notify_all()`), para que no queden esperando indefinidamente.
3. Se cierra el socket principal. 
4. Se hace `shutdown(SHUT_RDWR)` sobre cada conexión de cliente activa, para desbloquear cualquier lectura/escritura pendiente.
5. Se espera a que todos los hilos activos terminen (`thread.join`), respetando un **timeout global acotado** (`SHUTDOWN_TIMEOUT`) repartido entre todos los hilos según el tiempo restante hasta un `deadline` común, en lugar de un timeout fijo por hilo.

En el caso del cliente, este también maneja SIGTERM, pero de forma más simple que el servidor. Lo que hace es aprovechar que cerrar el socket corta cualquier operación de red activa.

Cuando llega la señal, una goroutine aparte llama a client.Close(), cerrando la conexión TCP. Eso hace que el Run() principal, que en ese momento está bloqueado enviando o recibiendo datos, falle inmediatamente y termine. Como se usa un context para saber si esa señal fue la que causo el corte, el programa distingue un cierre con SIGTERM de un error, y sale con código 0 en el primer caso y 1 en el segundo.

Nota: Apenas llega la señal, se corta la conexión y el programa termina.