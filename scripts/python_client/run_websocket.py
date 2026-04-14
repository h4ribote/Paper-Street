import os
import json
import argparse
import time
from client import PaperStreetClient

def get_client():
    base_url = os.environ.get("PAPERSTREET_BASE_URL", "http://localhost:8000")
    api_key = os.environ.get("PAPERSTREET_API_KEY")
    return PaperStreetClient(base_url, api_key)

def on_message(data):
    print("\n--- Message Received ---")
    print(json.dumps(data, indent=2, ensure_ascii=False))
    print("------------------------")

def main():
    parser = argparse.ArgumentParser(description="Test WebSocket Endpoints")
    parser.add_argument("topic", help="Topic to subscribe to (e.g. market.ticker, market.orderbook.1, user.orders)")
    parser.add_argument("--duration", type=int, default=10, help="How many seconds to listen")

    args = parser.parse_args()
    client = get_client()

    try:
        print(f"Connecting to WebSocket...")
        client.connect_ws()

        print(f"Registering callback and subscribing to: {args.topic}")
        client.on(args.topic, on_message)
        client.subscribe(args.topic)

        print(f"Listening for {args.duration} seconds. Press Ctrl+C to stop earlier.")
        for i in range(args.duration):
            time.sleep(1)

        print("Unsubscribing and disconnecting...")
        client.unsubscribe(args.topic)
        client.disconnect_ws()
        print("Done.")
    except KeyboardInterrupt:
        print("\nInterrupted by user. Disconnecting...")
        client.disconnect_ws()
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
