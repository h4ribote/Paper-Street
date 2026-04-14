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
    parser = argparse.ArgumentParser(description="Test Order Endpoints")
    parser.add_argument("action", choices=["list", "create", "cancel", "get"], help="Action to perform")
    parser.add_argument("--asset_id", type=int, default=1, help="Asset ID")
    parser.add_argument("--order_id", type=int, help="Order ID")
    parser.add_argument("--side", choices=["BUY", "SELL"], default="BUY", help="Order Side")
    parser.add_argument("--type", choices=["MARKET", "LIMIT", "STOP", "STOP_LIMIT"], default="LIMIT", help="Order Type")
    parser.add_argument("--quantity", type=int, default=10, help="Order Quantity")
    parser.add_argument("--price", type=float, help="Order Price")
    parser.add_argument("--leverage", type=int, default=1, help="Leverage")

    args = parser.parse_args()
    client = get_client()

    try:
        if args.action == "list":
            print_json(client.get_orders(asset_id=args.asset_id if args.asset_id != 1 else None))
        elif args.action == "create":
            if args.type in ["LIMIT", "STOP_LIMIT"] and args.price is None:
                print("Error: --price is required for LIMIT orders")
                return
            print_json(client.create_order(
                asset_id=args.asset_id,
                side=args.side,
                order_type=args.type,
                quantity=args.quantity,
                price=args.price,
                leverage=args.leverage
            ))
        elif args.action == "cancel":
            if not args.order_id:
                print("Error: --order_id is required")
                return
            print_json(client.cancel_order(args.order_id, args.asset_id))
        elif args.action == "get":
            if not args.order_id:
                print("Error: --order_id is required")
                return
            print_json(client.get_order(args.order_id, args.asset_id))
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
