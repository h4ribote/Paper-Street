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
    parser.add_argument("action", choices=["list", "get", "add_liquidity", "positions", "remove_liquidity", "swap", "margin_list", "margin_get", "margin_supply", "margin_withdraw", "margin_positions", "margin_topup", "margin_liquidations"], help="Action to perform")
    parser.add_argument("--pool_id", type=int, default=1, help="Pool ID")
    parser.add_argument("--base_amount", type=int, help="Base Amount")
    parser.add_argument("--quote_amount", type=int, help="Quote Amount")
    parser.add_argument("--lower_tick", type=int, help="Lower Tick")
    parser.add_argument("--upper_tick", type=int, help="Upper Tick")
    parser.add_argument("--position_id", type=int, help="Position ID")
    parser.add_argument("--from_currency", type=str, help="From Currency")
    parser.add_argument("--to_currency", type=str, help="To Currency")
    parser.add_argument("--amount", type=int, help="Amount")
    parser.add_argument("--cash_amount", type=int, help="Cash Amount")
    parser.add_argument("--asset_amount", type=int, help="Asset Amount")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "list":
            print_json(client.get_pools())
        elif args.action == "get":
            print_json(client.get_pool(args.pool_id))
        elif args.action == "add_liquidity":
            print_json(client.add_liquidity(args.pool_id, args.base_amount, args.quote_amount, args.lower_tick, args.upper_tick))
        elif args.action == "positions":
            print_json(client.get_pool_positions())
        elif args.action == "remove_liquidity":
            print_json(client.remove_liquidity(args.position_id))
        elif args.action == "swap":
            print_json(client.swap(args.pool_id, args.from_currency, args.to_currency, args.amount))
        elif args.action == "margin_list":
            print_json(client.get_margin_pools())
        elif args.action == "margin_get":
            print_json(client.get_margin_pool(args.pool_id))
        elif args.action == "margin_supply":
            print_json(client.supply_margin(args.pool_id, args.cash_amount, args.asset_amount))
        elif args.action == "margin_withdraw":
            print_json(client.withdraw_margin(args.pool_id, args.cash_amount, args.asset_amount))
        elif args.action == "margin_positions":
            print_json(client.get_margin_positions())
        elif args.action == "margin_topup":
            print_json(client.topup_margin(args.position_id, args.amount))
        elif args.action == "margin_liquidations":
            print_json(client.get_margin_liquidations())
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
