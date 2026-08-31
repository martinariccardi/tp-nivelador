import socket
import logger
import safe_socket
from . import protocol
from lottery import Lottery

class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.lottery = Lottery(storage_path)

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                client_message = protocol.deserialize(client_socket)
                if not client_message:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    return
                message_amount += 1
                # Mejorar
                if client_message["type"] == "END_BETS":
                    winners = self._choose_winners(client_message["data"])
                    self.send_winners(client_socket, winners)
                    return
                else:
                    self.lottery.store_bets([client_message["data"]])
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "messages-amount", message_amount
            )
            raise e

    def _choose_winners(self, agency_id):
        bets = self.lottery.load_bets()
        winners = []
        for bet in bets:
            if bet.agency_id == int(agency_id) and self.lottery.has_won(bet):
                winners.append(bet)
        return winners

    def send_winners(self, socket, winners):
        safe_socket.send_all(socket, protocol.serialize_winners(winners))

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_socket)
