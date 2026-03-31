#!/bin/sh
set -eu

: "${API_BASE_URL:?API_BASE_URL environment variable is required}"
: "${ADMIN_PASSWORD:?ADMIN_PASSWORD environment variable is required}"

pids=""

start_bot() {
  name="$1"
  shift
  echo "Starting bot: ${name}"
  env "$@" &
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

start_bot "market_maker" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=market_maker ./market_maker
start_bot "news_reactor" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=news_reactor ./news_reactor

start_bot "momentum_chaser_a" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=momentum_chaser_a ASSET_ID=101 ./momentum_chaser
start_bot "momentum_chaser_b" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=momentum_chaser_b ASSET_ID=102 ./momentum_chaser
start_bot "momentum_chaser_c" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=momentum_chaser_c ASSET_ID=103 ./momentum_chaser

start_bot "dip_buyer_a" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=dip_buyer_a ASSET_ID=101 ./dip_buyer
start_bot "dip_buyer_b" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=dip_buyer_b ASSET_ID=102 ./dip_buyer
start_bot "dip_buyer_c" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=dip_buyer_c ASSET_ID=103 ./dip_buyer

start_bot "reversal_sniper_a" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=reversal_sniper_a ASSET_ID=101 ./reversal_sniper
start_bot "reversal_sniper_b" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=reversal_sniper_b ASSET_ID=102 ./reversal_sniper
start_bot "reversal_sniper_c" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=reversal_sniper_c ASSET_ID=103 ./reversal_sniper

start_bot "grid_trader_a" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=grid_trader_a ASSET_ID=101 ./grid_trader
start_bot "grid_trader_b" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=grid_trader_b ASSET_ID=102 ./grid_trader
start_bot "grid_trader_c" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=grid_trader_c ASSET_ID=103 ./grid_trader

start_bot "whale_northern" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=whale_northern ASSET_ID=101 ./whale
start_bot "whale_oceanic" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=whale_oceanic ASSET_ID=102 ./whale
start_bot "whale_energy" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=whale_energy ASSET_ID=103 ./whale

start_bot "national_ai_arcadia" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_arcadia ./national_ai
start_bot "national_ai_boros" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_boros ./national_ai
start_bot "national_ai_el_dorado" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_el_dorado ./national_ai
start_bot "national_ai_neo_venice" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_neo_venice ./national_ai
start_bot "national_ai_san_verde" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_san_verde ./national_ai
start_bot "national_ai_novaya_zemlya" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_novaya_zemlya ./national_ai
start_bot "national_ai_pearl_river" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=national_ai_pearl_river ./national_ai

start_bot "arbitrageur" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=arbitrageur INDEX_ASSET_ID=201 COMPONENT_ASSET_IDS=101,102,103 ENABLE_FX_ARB=true ./arbitrageur
start_bot "yield_hunter" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=yield_hunter ./yield_hunter
start_bot "public_consumer" API_BASE_URL="$API_BASE_URL" ADMIN_PASSWORD="$ADMIN_PASSWORD" BOT_ROLE=public_consumer ./public_consumer

wait
