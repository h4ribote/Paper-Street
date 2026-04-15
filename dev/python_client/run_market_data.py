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
    parser = argparse.ArgumentParser(description="Test Market Data Endpoints")
    parser.add_argument("action", choices=["health", "assets", "asset", "orderbook", "ticker", "news"], help="Action to perform")
    parser.add_argument("--asset_id", type=int, default=1, help="Asset ID")
    parser.add_argument("--depth", type=int, default=20, help="Orderbook depth")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "health":
            print_json(client.check_health())
        elif args.action == "assets":
            print_json(client.get_assets())
        elif args.action == "asset":
            print_json(client.get_asset(args.asset_id))
        elif args.action == "orderbook":
            print_json(client.get_orderbook(args.asset_id, args.depth))
        elif args.action == "ticker":
            print_json(client.get_ticker())
        elif args.action == "news":
            print_json(client.get_news())
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
