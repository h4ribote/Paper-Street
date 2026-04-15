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
    parser = argparse.ArgumentParser(description="Test Companies Endpoints")
    parser.add_argument("action", choices=["capital", "financing", "buyback", "production", "supply_chain", "financials", "dividends", "simulate"], help="Action to perform")
    parser.add_argument("--company_id", type=int, help="Company ID")
    parser.add_argument("--target_amount", type=int, help="Target Amount")
    parser.add_argument("--reason", type=str, help="Reason")
    parser.add_argument("--budget", type=int, help="Budget")
    parser.add_argument("--limit", type=int, help="Limit")
    parser.add_argument("--quarters", type=int, default=1, help="Quarters to simulate")

    args = parser.parse_args()
    client = get_client()

    try:
        if not args.company_id:
            print("Error: --company_id is required")
            return

        if args.action == "capital":
            print_json(client.get_capital_structure(args.company_id))
        elif args.action == "financing":
            print_json(client.initiate_financing(args.company_id, args.target_amount, args.reason))
        elif args.action == "buyback":
            print_json(client.authorize_buyback(args.company_id, args.budget))
        elif args.action == "production":
            print_json(client.get_production_status(args.company_id))
        elif args.action == "supply_chain":
            print_json(client.get_supply_chain(args.company_id))
        elif args.action == "financials":
            print_json(client.get_financials(args.company_id, args.limit))
        elif args.action == "dividends":
            print_json(client.get_dividends(args.company_id, args.limit))
        elif args.action == "simulate":
            print_json(client.simulate_company(args.company_id, args.quarters))
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
