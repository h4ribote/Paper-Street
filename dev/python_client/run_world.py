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
    parser = argparse.ArgumentParser(description="Test World Meta Endpoints")
    parser.add_argument("action", choices=["season", "regions", "companies", "events", "leaderboard"], help="Action to perform")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "season":
            print_json(client.get_current_season())
        elif args.action == "regions":
            print_json(client.get_regions())
        elif args.action == "companies":
            print_json(client.get_companies())
        elif args.action == "events":
            print_json(client.get_events())
        elif args.action == "leaderboard":
            print_json(client.get_leaderboard())
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
