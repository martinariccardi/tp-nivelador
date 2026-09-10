import socket
import logger
import safe_socket
import threading
import time
from . import protocol
from lottery import Lottery

SHUTDOWN_TIMEOUT = 10.0

class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str, agency_quorum_min: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.lottery = Lottery(storage_path)
        self.lock = threading.Lock()
        self.agency_quorum_min = agency_quorum_min
        self.condition = threading.Condition(self.lock)
        self.finished_agencies = 0

        self.running = True
        self.client_connections = []
        self.active_threads = []
        self.socket = None

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        try:
            logger.info(action, logger.LogResult.in_progress)
            while self.running:
                try: 
                    client_message = protocol.deserialize(client_socket)
                except Exception as e: 
                    logger.error(action, logger.LogResult.fail, "malformed-batch", str(e))
                    self.send_nack(client_socket)
                    continue
                
                if not client_message:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    return

                message_amount += 1
                message_type = client_message["type"]

                if message_type == "END_BETS":
                    self._handle_end_bets(client_socket, client_message["data"])
                    return

                if message_type == "NEW_BATCH":
                    self._handle_new_batch(client_socket, client_message["data"])
                    continue

                raise ValueError(f"Tipo de mensaje no soportado: {message_type}")

        except Exception as e:
            logger.error(
                action,
                logger.LogResult.fail,
                "messages-amount",
                message_amount,
            )
            raise e
        finally:
            client_socket.close()

    def _handle_new_batch(self, client_socket, bets):
        with self.lock:
            self.lottery.store_bets(bets)
        self.send_ack(client_socket)

    def _handle_end_bets(self, client_socket, agency_id):
        with self.condition:
            self.finished_agencies += 1
            self.condition.notify_all()
            self.condition.wait_for(
                lambda: self.finished_agencies >= self.agency_quorum_min or not self.running
            )
        with self.lock:
            winners = self._choose_winners(agency_id)
        self.send_winners(client_socket, winners)

    def _choose_winners(self, agency_id):
        bets = self.lottery.load_bets()
        winners = []
        for bet in bets:
            if bet.agency_id == int(agency_id) and self.lottery.has_won(bet):
                winners.append(bet)
        return winners

    def send_winners(self, socket, winners):
        safe_socket.send_all(socket, protocol.serialize_winners(winners))

    def send_ack(self, socket):
        safe_socket.send_all(socket, protocol.serialize_ack())

    def send_nack(self, socket):
        safe_socket.send_all(socket, protocol.serialize_nack())

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            self.socket = server_socket
            while self.running:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                    self.client_connections.append(client_socket)
                except Exception as e:
                    if not self.running:
                        break
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                thread = threading.Thread(
                    target= self._handle_client,
                    args=(client_socket,)                   
                )

                self.active_threads.append(thread)

                thread.start()

    def handle_sigterm(self, _signum=None, _frame=None):
        self.running = False

        with self.condition:
            self.condition.notify_all()

        if self.socket is not None:
            try:
                self.socket.close()
            except OSError:
                pass

        for conn in self.client_connections:
            try: 
                conn.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass

        deadline = time.time() + SHUTDOWN_TIMEOUT
        for thread in self.active_threads:
            time_left = deadline - time.time()
            if time_left > 0:
                thread.join(timeout=time_left)
            else:
                break

                

   