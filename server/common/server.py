import signal
import socket
import logging
import sys

from common.utils import Bet, store_bets, load_bets, has_won


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._shutdown = False
        self._agencies_finished = set()
        self._sorteo_done = False
        self._ganadores_por_agencia = {}

    def graceful_shutdown(self, signum, frame):
        logging.info("action: shutdown | result: in_progress | msg: Closing server socket")
        self._server_socket.close()
        logging.info("action: shutdown | result: success | msg: Server socket closed")
        self._shutdown = True
        sys.exit(0)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        signal.signal(signal.SIGTERM, self.graceful_shutdown)
        signal.signal(signal.SIGINT, self.graceful_shutdown)

        while not self._shutdown:
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError:
                break

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
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
                self._agencies_finished.add(agency_id)
                logging.info(f"action: fin_agencia | result: success | agencia: {agency_id} | total_agencias: {len(self._agencies_finished)}")
                client_sock.sendall(b"OK\n")
                if len(self._agencies_finished) == 5 and not self._sorteo_done:
                    self._realizar_sorteo()
                return

            if msg.startswith("CONSULTA_GANADORES|"):
                agency_id = msg.split("|")[1]
                if not self._sorteo_done:
                    client_sock.sendall(b"ERROR\n")
                    logging.info(f"action: consulta_ganadores | result: fail | agencia: {agency_id} | msg: sorteo no realizado")
                    return
                ganadores = self._ganadores_por_agencia.get(int(agency_id), [])
                dni_list = "|".join(ganadores)
                client_sock.sendall((dni_list + "\n").encode('utf-8'))
                logging.info(f"action: consulta_ganadores | result: success | agencia: {agency_id} | cant_ganadores: {len(ganadores)}")
                return

            apuestas = msg.split('\n')
            bets = []
            for apuesta in apuestas:
                campos = apuesta.split('|')
                if len(campos) != 5:
                    raise ValueError("Apuesta inválida")
                bets.append(Bet('1', *campos))
            store_bets(bets)
            logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
            client_sock.sendall(b"OK\n")
        except Exception as e:
            logging.error(f'action: apuesta_recibida | result: fail | error: {e}')
            client_sock.sendall(b"ERROR\n")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
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
