import sys
import random

def generate_compose(num_clientes, outfile):
    with open(outfile, "w") as f:
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
        for i in range(1, num_clientes + 1):
            f.write(
                f"""  client{i}:
    container_name: client{i}
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID={i}
      - NOMBRE={random_name()}
      - APELLIDO={random_surname()}
      - DOCUMENTO={random_dni()}
      - NACIMIENTO={random_birth()}
      - NUMERO={random_number()}
    networks:
      - testing_net
    depends_on:
      - server
    volumes:
      - ./client/config.yaml:/config.yaml
      - ./.data/agency-{i}.csv:/agency.csv
"""
            )
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

def random_dni():
    return str(random.randint(20000000, 50000000))

def random_number():
    return str(random.randint(0000, 9999))

def random_name():
    return random.choice(["Santiago", "Lionel", "Maria", "Ana", "Carlos"])

def random_surname():
    return random.choice(["Lorca", "Perez", "Gomez", "Diaz", "Fernandez"])

def random_birth():
    year = random.randint(1950, 2005)
    month = random.randint(1, 12)
    day = random.randint(1, 28)
    return f"{year:04d}-{month:02d}-{day:02d}"

if __name__ == "__main__":
    num_clientes = int(sys.argv[1])
    outfile = sys.argv[2]
    generate_compose(num_clientes, outfile)