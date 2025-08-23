import sys

def generate_compose(num_clientes, outfile):
    with open(outfile, "w") as f:
        # Encabezado y servidor
        f.write(
            """name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
    networks:
      - testing_net
    volumes:
      - ./server/config.ini:/config.ini
"""
        )
        # Clientes
        for i in range(1, num_clientes + 1):
            f.write(
                f"""  client{i}:
    container_name: client{i}
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID={i}
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server
    volumes:
      - ./client/config.yaml:/config.yaml
"""
            )
        # Redes
        f.write(
            """
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
"""
        )

if __name__ == "__main__":
    num_clientes = int(sys.argv[1])
    outfile = sys.argv[2]
    generate_compose(num_clientes, outfile)