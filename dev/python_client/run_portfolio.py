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
    parser = argparse.ArgumentParser(description="Test Portfolio Endpoints")
    parser.add_argument("action", choices=["balances", "assets", "positions", "history", "performance"], help="Action to perform")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "balances":
            print_json(client.get_balances())
        elif args.action == "assets":
            print_json(client.get_portfolio_assets())
        elif args.action == "positions":
            print_json(client.get_positions())
        elif args.action == "history":
            print_json(client.get_history())
        elif args.action == "performance":
            print_json(client.get_performance())
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
