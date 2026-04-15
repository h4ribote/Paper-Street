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
    parser = argparse.ArgumentParser(description="Test Indices Endpoints")
    parser.add_argument("action", choices=["list", "get", "create", "redeem"], help="Action to perform")
    parser.add_argument("--asset_id", type=int, help="Index Asset ID")
    parser.add_argument("--quantity", type=int, default=1, help="Quantity")
    parser.add_argument("--user_id", type=int, help="User ID")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "list":
            print_json(client.get_indices())
        else:
            if not args.asset_id:
                print("Error: --asset_id is required")
                return

            if args.action == "get":
                print_json(client.get_index(args.asset_id))
            elif args.action == "create":
                print_json(client.create_index(args.asset_id, args.quantity, args.user_id))
            elif args.action == "redeem":
                print_json(client.redeem_index(args.asset_id, args.quantity, args.user_id))
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
