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
    parser = argparse.ArgumentParser(description="Test Bonds Endpoints")
    parser.add_argument("action", choices=["list", "get"], help="Action to perform")
    parser.add_argument("--bond_id", type=int, help="Bond ID")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "list":
            print_json(client.get_bonds())
        elif args.action == "get":
            if not args.bond_id:
                print("Error: --bond_id is required")
                return
            print_json(client.get_bond(args.bond_id))
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
