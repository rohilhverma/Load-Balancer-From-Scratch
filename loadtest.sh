#!/bin/zsh

BASE="$(printf '\150\164\164\160\072\057\057\154\157\143\141\154\150\157\163\164')"
PROXY="$BASE:8081/"
TOTAL=500
CLIENTS=50 # IPHASH: distinct X-Forwarded-For IPs, so each client sends TOTAL/CLIENTS requests

fail_server() {
  local port=$1
  local delay=$2

  sleep "$delay"
  echo "========== FAILING SERVER $port after ${delay}s =========="

  curl -sS "$BASE:$port/fail"
  echo
}

# Fail 8083 after 5-15 seconds.
delay_8083=$((5 + RANDOM % 11))

# Fail 8085 later, after 20-30 seconds.
delay_8085=$((20 + RANDOM % 11))

fail_server 8083 "$delay_8083" &
fail_pid_8083=$!

fail_server 8085 "$delay_8085" &
fail_pid_8085=$!

for i in {1..$TOTAL}; do
  (
    xff=()
    label=""
    client_ip="10.0.0.$(( i % CLIENTS ))"   # IPHASH
    xff=(-H "X-Forwarded-For: $client_ip")  # IPHASH
    label=" | ip: $client_ip"               # IPHASH
    started=$(date "+%Y-%m-%d %H:%M:%S")

    result=$(curl -sS -o /dev/null "${xff[@]}" \
      -w "HTTP %{http_code} | duration: %{time_total}s" \
      "$PROXY" 2>&1)

    exit_code=$?
    completed=$(date "+%Y-%m-%d %H:%M:%S")

    if [[ $exit_code -eq 0 ]]; then
      echo "Request $i$label | started: $started | completed: $completed | $result"
    else
      echo "Request $i$label | FAILED | started: $started | completed: $completed | $result"
    fi
  ) &

  sleep 0.1
done

wait "$fail_pid_8083"
wait "$fail_pid_8085"
wait

echo "========== LOAD TEST COMPLETE =========="
