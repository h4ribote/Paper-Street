#!/bin/sh
set -eu

: "${API_BASE_URL:?API_BASE_URL environment variable is required}"
: "${ADMIN_PASSWORD:?ADMIN_PASSWORD environment variable is required}"

pids=""

start_bot() {
  name="$1"
  cmd="$2"
  shift 2
  echo "Starting bot: ${name}"
  env "$@" "$cmd" &
  pids="$pids $!"
}

terminate() {
  if [ -n "$pids" ]; then
    echo "Stopping bots..."
    for pid in $pids; do
      kill "$pid" 2>/dev/null || true
    done
  fi
}

trap terminate INT TERM

start_bot "market_maker" ./market_maker API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=market_maker
start_bot "news_reactor" ./news_reactor API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=news_reactor

start_bot "momentum_chaser_a" ./momentum_chaser API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=momentum_chaser_a ASSET_ID=101
start_bot "momentum_chaser_b" ./momentum_chaser API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=momentum_chaser_b ASSET_ID=102
start_bot "momentum_chaser_c" ./momentum_chaser API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=momentum_chaser_c ASSET_ID=103

start_bot "dip_buyer_a" ./dip_buyer API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=dip_buyer_a ASSET_ID=101
start_bot "dip_buyer_b" ./dip_buyer API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=dip_buyer_b ASSET_ID=102
start_bot "dip_buyer_c" ./dip_buyer API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=dip_buyer_c ASSET_ID=103

start_bot "reversal_sniper_a" ./reversal_sniper API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=reversal_sniper_a ASSET_ID=101
start_bot "reversal_sniper_b" ./reversal_sniper API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=reversal_sniper_b ASSET_ID=102
start_bot "reversal_sniper_c" ./reversal_sniper API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=reversal_sniper_c ASSET_ID=103

start_bot "grid_trader_a" ./grid_trader API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=grid_trader_a ASSET_ID=101
start_bot "grid_trader_b" ./grid_trader API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=grid_trader_b ASSET_ID=102
start_bot "grid_trader_c" ./grid_trader API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=grid_trader_c ASSET_ID=103

start_bot "whale_northern" ./whale API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=whale_northern ASSET_ID=101
start_bot "whale_oceanic" ./whale API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=whale_oceanic ASSET_ID=102
start_bot "whale_energy" ./whale API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=whale_energy ASSET_ID=103

start_bot "national_ai_arcadia" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_arcadia
start_bot "national_ai_boros" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_boros
start_bot "national_ai_el_dorado" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_el_dorado
start_bot "national_ai_neo_venice" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_neo_venice
start_bot "national_ai_san_verde" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_san_verde
start_bot "national_ai_novaya_zemlya" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_novaya_zemlya
start_bot "national_ai_pearl_river" ./national_ai API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_pearl_river

start_bot "arbitrageur" ./arbitrageur API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=arbitrageur INDEX_ASSET_ID=201 COMPONENT_ASSET_IDS=101,102,103 ENABLE_FX_ARB=true
start_bot "yield_hunter" ./yield_hunter API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=yield_hunter
start_bot "public_consumer" ./public_consumer API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=public_consumer

wait $pids
