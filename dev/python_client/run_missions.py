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
    parser = argparse.ArgumentParser(description="Test Missions Endpoints")
    parser.add_argument("action", choices=["rank", "daily", "user", "complete"], help="Action to perform")
    parser.add_argument("--mission_id", type=int, help="Mission ID")
    parser.add_argument("--user_id", type=int, help="User ID")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "rank":
            print_json(client.get_rank(args.user_id))
        elif args.action == "daily":
            print_json(client.get_daily_missions(args.user_id))
        elif args.action == "user":
            print_json(client.get_user_missions())
        elif args.action == "complete":
            if not args.mission_id:
                print("Error: --mission_id is required")
                return
            print_json(client.complete_mission(args.mission_id, args.user_id))
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
