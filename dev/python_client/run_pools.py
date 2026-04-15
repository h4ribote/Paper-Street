import os
import json
import argparse
from client import PaperStreetClient

def get_client():
    base_url = os.environ.get("PAPERSTREET_BASE_URL", "http://localhost:8000")
    api_key = os.environ.get("PAPERSTREET_API_KEY")
    return PaperStreetClient(base_url, api_key)

def print_json(data):
    print(json.dumps(data, indent=2, ensure_ascii=False))

def main():
    parser = argparse.ArgumentParser(description="Test Pools Endpoints")
    parser.add_argument("action", choices=["list", "get", "margin_list", "margin_get"], help="Action to perform")
    parser.add_argument("--pool_id", type=int, default=1, help="Pool ID")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "list":
            print_json(client.get_pools())
        elif args.action == "get":
            print_json(client.get_pool(args.pool_id))
        elif args.action == "margin_list":
            print_json(client.get_margin_pools())
        elif args.action == "margin_get":
            print_json(client.get_margin_pool(args.pool_id))
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
