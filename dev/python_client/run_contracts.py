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
    parser = argparse.ArgumentParser(description="Test Contracts Endpoints")
    parser.add_argument("action", choices=["list", "get", "deliver", "user"], help="Action to perform")
    parser.add_argument("--contract_id", type=int, help="Contract ID")
    parser.add_argument("--quantity", type=int, default=1, help="Quantity to deliver")
    parser.add_argument("--user_id", type=int, help="User ID")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "list":
            print_json(client.get_contracts(args.user_id))
        elif args.action == "get":
            if not args.contract_id:
                print("Error: --contract_id is required")
                return
            print_json(client.get_contract(args.contract_id, args.user_id))
        elif args.action == "deliver":
            if not args.contract_id:
                print("Error: --contract_id is required")
                return
            print_json(client.deliver_contract(args.contract_id, args.quantity, args.user_id))
        elif args.action == "user":
            print_json(client.get_user_contracts())
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
