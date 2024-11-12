import json
import logging
import os
import time

import dotenv
import jwt
import requests

dotenv.load_dotenv()

logging.basicConfig(
    level=logging.DEBUG, format="%(asctime)s - %(levelname)s - %(message)s"
)

CIPHEROWL_TOKEN_PATH = os.path.expanduser("~/.cipherowl/token-cache.json")
CIPHEROWL_API_URL = "https://svc.cipherowl.ai"
CLIENT_ID = os.getenv("CLIENT_ID")
CLIENT_SECRET = os.getenv("CLIENT_SECRET")


def get_token_from_cache():
    if os.path.exists(CIPHEROWL_TOKEN_PATH):
        with open(CIPHEROWL_TOKEN_PATH, "r") as f:
            token_cache = json.load(f)
            token = token_cache.get("access_token")

            # ensure token is not expired
            decoded = jwt.decode(token, options={"verify_signature": False})
            if time.time() < decoded["exp"]:
                logging.debug("Get token from cache")
                return token
    return None


def write_token_to_cache(token):
    os.makedirs(os.path.dirname(CIPHEROWL_TOKEN_PATH), exist_ok=True)
    with open(CIPHEROWL_TOKEN_PATH, "w") as f:
        json.dump({"access_token": token}, f)
    logging.debug("Write token to cache")


def get_token_from_server():
    url = f"{CIPHEROWL_API_URL}/oauth/token"

    payload = json.dumps(
        {
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "audience": "svc.cipherowl.ai",
            "grant_type": "client_credentials",
        }
    )
    headers = {"Content-Type": "application/json"}

    response = requests.request("POST", url, headers=headers, data=payload)
    response.raise_for_status()

    token = response.json()["access_token"]
    logging.debug("Get token from server")
    write_token_to_cache(token)
    return token


def get_token():
    # use token cache to improve performance and reduce server load
    token = get_token_from_cache()
    if token:
        return token

    token = get_token_from_server()
    return token


def main():
    project = "partner"
    url = f"{CIPHEROWL_API_URL}/api/v1/sanction?project={project}&chain=bitcoin_mainnet&address=12udabs2TkX7NXCSj6KpqXfakjE52ZPLhz"

    token = get_token()
    headers = {"Authorization": f"Bearer {token}"}

    response = requests.request("GET", url, headers=headers)
    response.raise_for_status()
    print(json.dumps(response.json(), indent=4))


if __name__ == "__main__":
    main()
