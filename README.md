## Ejercicio 1: Generación dinámica de Docker Compose

Para el Ejercicio N°1, se desarrolló una solución que permite crear un archivo de definición de Docker Compose con una cantidad configurable de clientes, siguiendo el formato `client1`, `client2`, ..., `clientN`. El archivo también genera al servidor y la sección de networks (En otras palabras, escribe el archivo compose en su totalidad).

### Scripts utilizados

- **`generar-compose.sh`**: Script principal en Bash que recibe dos parámetros: el nombre del archivo de salida y la cantidad de clientes a generar. Este script invoca un script auxiliar en Python para generar el contenido del archivo Docker Compose.
- **`generate_clients.py`**: Script en Python que recibe la cantidad de clientes y el nombre del archivo de salida, y genera el archivo Docker Compose con la definición del servidor y los clientes.

### Uso

Para generar el archivo de Docker Compose con, por ejemplo, 5 clientes, se debe ejecutar:

```sh
[generar-compose.sh](http://_vscodecontentref_/0) [docker-compose-dev.yaml](http://_vscodecontentref_/1) 5
```


## Ejercicio 2: Configuración dinámica con Docker Volumes

### Descripción

Para cumplir con el requerimiento de **Ejercicio N°2**, se modificó la arquitectura de cliente y servidor para que los archivos de configuración (`config.ini` para el servidor y `config.yaml` para el cliente) **no estén embebidos en la imagen Docker**, sino que sean **inyectados como volúmenes** al momento de levantar los contenedores.

Esto permite que cualquier cambio realizado en los archivos de configuración **se refleje inmediatamente en la ejecución**, sin necesidad de reconstruir las imágenes Docker.

---

### Implementación

#### 1. **Docker Compose**

En el archivo `docker-compose-dev.yaml` (y el generador de compose), se agregaron los volúmenes para inyectar los archivos de configuración:

```yaml
  server:
    ...
    volumes:
      - ./server/config.ini:/config.ini

  client1:
    ...
    volumes:
      - ./client/config.yaml:/config.yaml
```

Cada cliente y el servidor reciben su archivo de configuración desde el host, montado en el path `/config.yaml` o `/config.ini` dentro del contenedor.

#### 2. **Dockerfile**

En los Dockerfile de cliente y servidor, **no se copia el archivo de configuración** durante el build.  
Esto se logra ignorando los archivos en `.dockerignore`:

```ignore
server/config.ini
client/config.yaml
```

### Ventajas

- **Cambios inmediatos:** Modificar los archivos de configuración en el host impacta directamente en los contenedores en la próxima ejecución.
- **No requiere rebuild:** No es necesario reconstruir las imágenes Docker para aplicar cambios de configuración.
- **Persistencia:** Los archivos de configuración se mantienen fuera de la imagen y pueden versionarse o editarse fácilmente.


## Ejercicio 3: Validación del Echo Server con Netcat en Docker

Para el Ejercicio N°3 se desarrolló un script de bash llamado `validar-echo-server.sh` que permite verificar el correcto funcionamiento del servidor tipo echo utilizando el comando `netcat` (nc) **dentro de los contenedores Docker**, sin instalar nada en la máquina host ni exponer puertos.


### Implementación

El script `validar-echo-server.sh` se encuentra en la raíz del proyecto.  
Su funcionamiento es el siguiente:

1. Define el mensaje de prueba a enviar (`test_echo`).
2. Utiliza el comando `docker exec` para ejecutar `nc` dentro de un contenedor cliente, conectándose al servidor por nombre de red interna (`server:12345`).
3. Envía el mensaje y recibe la respuesta.
4. Compara la respuesta con el mensaje original.
5. Imprime el resultado en el formato:
   - `action: test_echo_server | result: success` si la respuesta es correcta.
   - `action: test_echo_server | result: fail` si no lo es.

### Ejemplo de uso

```bash
[validar-echo-server.sh](http://_vscodecontentref_/0)
```


## Ejercicio 4: Finalización graceful ante SIGTERM

### Descripción

En este ejercicio se modificaron tanto el **servidor** (Python) como el **cliente** (Go) para que ambos sistemas finalicen de forma **graceful** al recibir la señal SIGTERM. Esto significa que, al momento de apagar los contenedores (por ejemplo, con `docker compose down`), todos los recursos abiertos (sockets, archivos, threads) se cierran correctamente antes de que el proceso principal termine.

---

### Implementación

#### Servidor (Python)

- Se agregó un método `graceful_shutdown` en la clase `Server` (`server/common/server.py`).
- Este método se registra como handler para las señales SIGTERM y SIGINT usando el módulo `signal`.
- Al recibir la señal, el servidor:
  - Cierra el socket principal del servidor.
  - Loguea el inicio y el éxito del cierre del socket.
  - Cambia el flag interno `_shutdown` para detener el loop principal.
  - Finaliza el proceso con `sys.exit(0)`.

#### Cliente (Go)

- En el cliente, se agregó un handler para las señales SIGTERM y SIGINT en el archivo `main.go`.
- Se utiliza el paquete `os/signal` para capturar estas señales.
- Cuando el cliente recibe una señal de terminación:
  - Se llama a una función de cierre (`CloseClient`) que libera el socket y cualquier recurso abierto.
  - Se imprime un mensaje en el log indicando el inicio y el éxito del cierre del recurso.
  - Finalmente, el proceso termina con `os.Exit(0)`.



## Ejercicio 5: Registro de apuestas de quiniela y comunicación robusta

### Descripción

En este ejercicio se modificó la lógica de negocio de cliente y servidor para simular el registro de apuestas de quiniela, cumpliendo con los requisitos de comunicación y persistencia definidos en la consigna.

---

### Solución

#### Cliente

- El cliente representa una agencia de quiniela y toma los datos de la apuesta (nombre, apellido, DNI, nacimiento, número) desde **variables de entorno**.
- Estos datos se serializan en un string con formato `nombre|apellido|documento|nacimiento|numero\n` mediante la función `serializeBet()`.
- El cliente envía la apuesta al servidor usando sockets TCP, asegurando el envío completo del mensaje (evitando short write).
- Al recibir la confirmación del servidor, imprime en el log:  
  `action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Servidor

- El servidor recibe la apuesta, la deserializa y la almacena usando la función provista `store_bets(...)`.
- Al persistir la apuesta, imprime en el log:  
  `action: apuesta_almacenada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

---

### Comunicación y diseño

#### 1. **Definición de un protocolo para el envío de los mensajes**
- Se definió un protocolo simple: cada apuesta se envía como una línea de texto con los campos separados por `|` y terminada en `\n`.
- El servidor espera ese formato y lo procesa en consecuencia.

#### 2. **Serialización de los datos**
- El cliente serializa los datos de la apuesta en el formato definido antes de enviarlos.
- El servidor deserializa los datos recibidos para crear el objeto `Bet`.

#### 3. **Correcta separación de responsabilidades entre modelo de dominio y capa de comunicación**
- El cliente tiene funciones separadas para la obtención de datos (`getEnv`), serialización (`serializeBet`), y comunicación por socket (`StartClientLoop`).
- El servidor separa la lógica de recepción de mensajes, deserialización y persistencia (`__handle_client_connection` y `store_bets`).

#### 4. **Correcto empleo de sockets, incluyendo manejo de errores y evitando los fenómenos conocidos como short read y short write**
- **Short write:** El cliente envía el mensaje en un bucle, asegurando que todos los bytes sean transmitidos.
- **Short read:** El servidor lee en un bucle hasta recibir el caracter de fin de línea (`\n`), asegurando que el mensaje esté completo antes de procesarlo.
- Ambos lados manejan errores de conexión y cierran los sockets correctamente.



## Ejercicio 6: Procesamiento por batchs de apuestas

### Descripción

En este ejercicio se modificó el cliente y el servidor para implementar el envío y procesamiento de apuestas en modalidad **batch** (chunks), permitiendo que cada cliente registre varias apuestas en una sola consulta al servidor. Esto mejora la eficiencia en la transmisión y el procesamiento de datos.

---

### Implementación

#### Cliente

- Cada cliente lee las apuestas desde su archivo CSV correspondiente (`.data/agency-{N}.csv`), que es inyectado en el contenedor mediante un volumen de Docker.
- Las apuestas se agrupan en **batchs** antes de ser enviadas al servidor. La cantidad máxima de apuestas por batch es configurable en `config.yaml` bajo la clave `batch: maxAmount`.
- Además, se controla que el tamaño de cada batch no exceda los **8kB** (8192 bytes).
- El cliente serializa cada batch en un string, separando las apuestas por saltos de línea y los campos por `|`.
- El envío de cada batch se realiza asegurando que todos los bytes sean transmitidos (evitando short write).
- Al recibir la respuesta del servidor, el cliente imprime en el log:
  - `action: apuesta_enviada | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}` si el batch fue aceptado.
  - `action: apuesta_enviada | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}` si hubo algún error.

#### Servidor

- El servidor recibe el batch, lo deserializa y procesa todas las apuestas.
- Si **todas** las apuestas del batch son válidas y se almacenan correctamente, responde con éxito y loguea:
  - `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`
- Si alguna apuesta es inválida, responde con un código de error y loguea:
  - `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`
- El procesamiento y almacenamiento de apuestas se realiza usando la función provista `store_bets(...)`.

---

### Configuración y Volúmenes

- Los archivos de apuestas `.data/agency-{N}.csv` se inyectan en cada contenedor cliente usando volúmenes de Docker, permitiendo modificar los datos sin reconstruir la imagen.
- La cantidad máxima de apuestas por batch se configura en `client/config.yaml` y puede ser ajustada para cumplir con el límite de 8kB por paquete.





## Ejercicio 7: Notificación de fin de apuestas y consulta de ganadores

### Descripción

En este ejercicio se modificó la lógica de cliente y servidor para coordinar el sorteo de la quiniela. Los clientes notifican al servidor cuando terminan de enviar todas sus apuestas, y luego consultan la lista de ganadores de su agencia. El servidor realiza el sorteo solo cuando todas las agencias han finalizado el envío de apuestas, y responde a cada cliente con los ganadores correspondientes.

---

### Implementación

#### Cliente

- Al finalizar el envío de todos los batchs de apuestas, el cliente envía un mensaje de notificación al servidor:  
  `FIN|<agency_id>\n`
- Inmediatamente después, el cliente consulta la lista de ganadores de su agencia enviando:  
  `CONSULTA_GANADORES|<agency_id>\n`
- Si el sorteo aún no se realizó, el servidor responde con `"ERROR"`, y el cliente reintenta la consulta hasta obtener una respuesta válida.
- Cuando recibe la lista de ganadores (DNIs separados por `|` o vacío si no hay ganadores), el cliente imprime en el log:  
  `action: consulta_ganadores | result: success | cant_ganadores: ${CANT}`

#### Servidor

- El servidor mantiene un registro de las agencias que notificaron el fin de envío de apuestas.
- Cuando recibe la notificación de las 5 agencias, realiza el sorteo usando las funciones provistas `load_bets(...)` y `has_won(...)`.
- El sorteo se loguea como:  
  `action: sorteo | result: success`
- Al recibir una consulta de ganadores antes del sorteo, responde con `"ERROR"` y loguea el intento.
- Una vez realizado el sorteo, responde a cada consulta con la lista de DNIs ganadores correspondientes a la agencia solicitante, sin hacer broadcast a todas las agencias.
- El servidor garantiza que no se responde ninguna consulta de ganadores con información parcial.

---

### Protocolo y sincronización

- La coordinación entre clientes y servidor se realiza mediante mensajes explícitos (`FIN|...` y `CONSULTA_GANADORES|...`).
- El servidor espera la notificación de todas las agencias antes de realizar el sorteo y responder consultas.
- Cada cliente recibe únicamente los ganadores de su propia agencia.

---

### Logs

- El cliente loguea el resultado de la consulta de ganadores con la cantidad obtenida.
- El servidor loguea el sorteo y cada consulta de ganadores, indicando éxito o estado de espera.



## Ejercicio 8: Concurrencia y sincronización en el servidor

### Descripción

En este ejercicio se modificó el servidor Python para aceptar conexiones y procesar mensajes en paralelo, permitiendo que múltiples clientes interactúen simultáneamente con la central de Lotería Nacional.

---

### Solución implementada

#### Procesamiento concurrente

- El servidor utiliza el módulo estándar `threading` para crear un nuevo thread por cada conexión entrante.
- Cada vez que un cliente se conecta, el servidor lanza un thread dedicado para procesar los mensajes de ese cliente, permitiendo que varias agencias envíen apuestas y consulten ganadores al mismo tiempo.

#### Sincronización y persistencia

- Para evitar condiciones de carrera y garantizar la integridad de los datos, se emplea un **lock** (`threading.Lock`) y una **condition variable** (`threading.Condition`) para proteger las operaciones críticas:
  - Acceso y modificación de la lista de agencias finalizadas.
  - Persistencia y lectura de apuestas mediante las funciones `store_bets` y `load_bets`.
  - Coordinación del sorteo y la consulta de ganadores.
- Los threads que consultan ganadores esperan en la condition variable hasta que el sorteo se haya realizado, asegurando que no se entregue información parcial.

#### Cierre graceful

- El servidor mantiene una lista de threads activos y, al recibir una señal de cierre (`SIGTERM` o `SIGINT`), espera que todos los threads finalicen correctamente antes de cerrar el proceso principal.

---

### Consideraciones sobre el GIL en CPython

> **Comentario sobre el GIL:**

La implementación de concurrencia en Python se ve afectada por el **Global Interpreter Lock (GIL)**, que impide que múltiples threads ejecuten bytecode de Python al mismo tiempo.  
Esto significa que, aunque el servidor puede aceptar y procesar conexiones en paralelo (especialmente operaciones de I/O como sockets y archivos), el rendimiento en tareas intensivas de CPU puede verse limitado en sistemas multiprocesador.

Sin embargo, para este caso de uso, donde la mayor parte del trabajo es I/O (espera de conexiones, lectura/escritura de archivos), el impacto del GIL es mínimo y la solución basada en threads es suficiente y segura.  
Se utilizaron locks explícitos para proteger la persistencia y los datos compartidos, ya que las funciones de almacenamiento y lectura de apuestas **no son thread-safe** por sí mismas.

