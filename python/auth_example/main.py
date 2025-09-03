import json
import urllib.parse
import logging
import os
import time

import dotenv
import jwt
import requests
from collections import namedtuple

dotenv.load_dotenv()

logging.basicConfig(
    level=logging.DEBUG, format="%(asctime)s - %(levelname)s - %(message)s"
)


TokenCache = namedtuple("TokenCache", ["access_token", "expires_at"])


class TokenManager:
    def __init__(self, base_url: str, client_id: str, client_secret: str):
        self.base_url = urllib.parse.urlparse(base_url)
        self.client_id = client_id
        self.client_secret = client_secret
        self.token_cache = None

    def _get_token_from_cache(self):
        if self.token_cache is None:
            return None

        if time.time() > self.token_cache.expires_at:
            return None

        logging.debug("Get token from cache")
        return self.token_cache.access_token

    def _write_token_to_cache(self, token):
        decoded = jwt.decode(token, options={"verify_signature": False})
        expires_at = decoded["exp"]
        self.token_cache = TokenCache(access_token=token, expires_at=expires_at)
        logging.debug("Write token to cache")

    def _get_token_from_server(self):
        url = urllib.parse.urlunparse(self.base_url._replace(path="/oauth/token"))

        payload = json.dumps(
            {
                "client_id": self.client_id,
                "client_secret": self.client_secret,
                "audience": self.base_url.netloc,
                "grant_type": "client_credentials",
            }
        )
        headers = {"Content-Type": "application/json"}

        response = requests.request("POST", url, headers=headers, data=payload)
        response.raise_for_status()

        token = response.json()["access_token"]
        logging.debug("Get token from server")
        self._write_token_to_cache(token)
        return token

    def get_token(self):
        # use token cache to improve performance and reduce server load
        token = self._get_token_from_cache()
        if token:
            return token

        token = self._get_token_from_server()
        return token


def main():

    cipherowl_api_url = "https://svc.cipherowl.ai"
    client_id = os.getenv("CLIENT_ID")
    client_secret = os.getenv("CLIENT_SECRET")

    token_manager = TokenManager(cipherowl_api_url, client_id, client_secret)
    token = token_manager.get_token()
    headers = {"Authorization": f"Bearer {token}"}

    url = f"{cipherowl_api_url}/api/screen/v1/chains/evm/addresses/0xf4377eda661e04b6dda78969796ed31658d602d4?config=co-high_risk_hops_2"
    response = requests.request("GET", url, headers=headers)
    response.raise_for_status()
    print(json.dumps(response.json(), indent=4))


if __name__ == "__main__":
    main()
