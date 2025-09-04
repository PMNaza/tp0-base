import threading
import signal
import socket
import logging
import sys
import os

from common.utils import Bet, store_bets, load_bets, has_won

class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._total_agencies = int(os.getenv("TOTAL_AGENCIES", "5"))
        self._shutdown = False
        self._agencies_finished = set()
        self._sorteo_done = False
        self._ganadores_por_agencia = {}
        self._lock = threading.Lock()
        self._condvar = threading.Condition(self._lock)
        self._threads = []

    def graceful_shutdown(self, signum, frame):
        logging.info("action: shutdown | result: in_progress | msg: Closing server socket")
        self._server_socket.close()
        logging.info("action: shutdown | result: success | msg: Server socket closed")
        self._shutdown = True
        for t in self._threads:
            t.join(timeout=5)
        sys.exit(0)

    def run(self):
        signal.signal(signal.SIGTERM, self.graceful_shutdown)
        signal.signal(signal.SIGINT, self.graceful_shutdown)

        while not self._shutdown:
            try:
                client_sock = self.__accept_new_connection()
                t = threading.Thread(target=self.__handle_client_connection, args=(client_sock,))
                t.start()
                self._threads.append(t)
            except OSError:
                break

    def __handle_client_connection(self, client_sock):
        try:
            data = b''
            while not data.endswith(b'\n'):
                chunk = client_sock.recv(4096)
                if not chunk:
                    break
                data += chunk
            msg = data.decode('utf-8').rstrip('\n')

            if msg.startswith("FIN|"):
                agency_id = msg.split("|")[1]
                with self._condvar:
                    self._agencies_finished.add(agency_id)
                    logging.info(f"action: fin_agencia | result: success | agencia: {agency_id} | total_agencias: {len(self._agencies_finished)}")
                    client_sock.sendall(b"OK\n")
                    if len(self._agencies_finished) == self._total_agencies and not self._sorteo_done:
                        self._realizar_sorteo()
                        self._condvar.notify_all()
                return

            if msg.startswith("CONSULTA_GANADORES|"):
                agency_id = msg.split("|")[1]
                with self._condvar:
                    while not self._sorteo_done:
                        self._condvar.wait()
                    ganadores = self._ganadores_por_agencia.get(int(agency_id), [])
                    dni_list = "|".join(ganadores)
                    client_sock.sendall((dni_list + "\n").encode('utf-8'))
                    logging.info(f"action: consulta_ganadores | result: success | agencia: {agency_id} | cant_ganadores: {len(ganadores)}")
                return

            apuestas = msg.split('\n')
            bets = []
            for apuesta in apuestas:
                campos = apuesta.split('|')
                if len(campos) != 6:
                    raise ValueError("Apuesta inválida")
                bets.append(Bet(*campos))
            with self._lock:
                store_bets(bets)
                logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
                client_sock.sendall(b"OK\n")
        except Exception as e:
            logging.error(f'action: apuesta_recibida | result: fail | error: {e}')
            client_sock.sendall(b"ERROR\n")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def _realizar_sorteo(self):
        self._ganadores_por_agencia = {}
        for bet in load_bets():
            if has_won(bet):
                ag = bet.agency
                if ag not in self._ganadores_por_agencia:
                    self._ganadores_por_agencia[ag] = []
                self._ganadores_por_agencia[ag].append(bet.document)
        self._sorteo_done = True
        logging.info("action: sorteo | result: success")
