#!/usr/bin/env bash
# comment-watch: author=ata comments and re:<id> replies. No live hcom.
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
LIB=$ROOT/.systemd-ops/lib/comment-watch
DRIVER=$ROOT/.systemd-ops/drivers/comment-watch
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
[[ -r $LIB ]] || fail "missing $LIB"
[[ -x $DRIVER ]] || fail "missing driver $DRIVER"
command -v sqlite3 >/dev/null || fail "sqlite3 required"
command -v jq >/dev/null || fail "jq required"

# shellcheck source=/dev/null
source "$LIB"

plant_board() {
  local root=$1
  local lead=$2
  mkdir -p "$root/.beads" "$root/.systemd-ops"
  sqlite3 "$root/.beads/beads.db" <<'SQL'
CREATE TABLE issues (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  issue_type TEXT NOT NULL DEFAULT 'task'
);
CREATE TABLE comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id TEXT NOT NULL,
  author TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
SQL
  if [[ -n $lead ]]; then
    cat >"$root/.systemd-ops/scope.toml" <<EOF
[scope]
id = "probe"
owned = ["ata-*"]

[coordination]
lead = "hcom:$lead"
EOF
  fi
}

add_issue() {
  sqlite3 "$1/.beads/beads.db" \
    "INSERT INTO issues(id, title, issue_type) VALUES('$2','$3','$4');"
}

add_comment() {
  sqlite3 "$1/.beads/beads.db" \
    "INSERT INTO comments(issue_id, author, text) VALUES('$2','$3','$4');"
}

last_id() {
  sqlite3 -noheader "$1/.beads/beads.db" "SELECT MAX(id) FROM comments;"
}

fake_notify=$TMP/system-notify
cat >"$fake_notify" <<'EOF'
#!/usr/bin/env bash
system_notify() {
  jq -nc \
    --arg root "${SYSTEMD_OPS_SCOPE_ROOT:-}" \
    --arg src "$1" \
    --arg intent "$2" \
    --arg delivery "${SYSTEM_NOTIFY_DELIVERY:-}" \
    --arg recipient "${SYSTEM_NOTIFY_RECIPIENT:-}" \
    --arg msg "$3" \
    '{root:$root,src:$src,intent:$intent,delivery:$delivery,recipient:$recipient,msg:$msg}' >>"$NOTIFY_LOG"
  return 0
}
EOF

NOTIFY_LOG=$TMP/notify.log
: >"$NOTIFY_LOG"
export NOTIFY_LOG
export SYSTEM_NOTIFY_LIB=$fake_notify
WAKE_LOG=$TMP/wakes.log
: >"$WAKE_LOG"
export COMMENT_WATCH_RECORD=$WAKE_LOG
export STATE_DIR=$TMP/state

BOARD=$TMP/board-a
plant_board "$BOARD" midi
add_issue "$BOARD" "omp-demo" "Fix the watch" epic
add_comment "$BOARD" "omp-demo" sf "old sf comment"
add_comment "$BOARD" "omp-demo" ata "historical ata, must not fire on first tick"
export ATA_COMMENT_BOARDS=$BOARD

ata_comment_watch_run || fail "first tick failed"
[[ -s $STATE_DIR/cursors.json ]] || fail "first tick wrote no cursor"
[[ ! -s $NOTIFY_LOG ]] || fail "first tick sent a notification: $(cat "$NOTIFY_LOG")"
[[ ! -s $WAKE_LOG ]] || fail "first tick recorded a wake: $(cat "$WAKE_LOG")"
max=$(last_id "$BOARD")
got=$(jq -r --arg db "$BOARD/.beads/beads.db" '.[$db]' "$STATE_DIR/cursors.json")
[[ $got == "$max" ]] || fail "first tick cursor $got want $max"

add_comment "$BOARD" "omp-demo" sf "sf must stay silent"
ata_comment_watch_run || fail "sf tick failed"
[[ ! -s $NOTIFY_LOG ]] || fail "sf comment notified: $(cat "$NOTIFY_LOG")"
[[ ! -s $WAKE_LOG ]] || fail "sf comment recorded a wake: $(cat "$WAKE_LOG")"


add_comment "$BOARD" "omp-demo" ata "please look at this"
ata_comment_watch_run || fail "ata tick failed"
[[ -s $NOTIFY_LOG ]] || fail "ata comment did not notify"
lines=$(jq -s 'length' "$NOTIFY_LOG")
[[ $lines -eq 1 ]] || fail "expected 1 notify, got $lines: $(cat "$NOTIFY_LOG")"
want=$'operator ata has commented on your epic omp-demo | Fix the watch:\n> please look at this\n\n'
jq -e --arg want "$want" '.msg == $want' "$NOTIFY_LOG" >/dev/null \
  || fail "payload mismatch: $(jq -j -r '"got="+(.msg|tojson)' "$NOTIFY_LOG")"
jq -e '.msg | endswith("\n")' "$NOTIFY_LOG" >/dev/null \
  || fail "payload missing trailing newline"
scope=$(jq -r '.root' "$NOTIFY_LOG")
[[ $scope == "$BOARD" ]] || fail "notify scope $scope want $BOARD"
src=$(jq -r '.src' "$NOTIFY_LOG")
[[ $src == mardi-gras:comment-watch ]] || fail "source $src"
intent=$(jq -r '.intent' "$NOTIFY_LOG")
[[ $intent == inform ]] || fail "intent $intent"
delivery=$(jq -r '.delivery' "$NOTIFY_LOG")
[[ $delivery == steer ]] || fail "delivery $delivery want steer"
[[ $(jq -r '.recipient' "$NOTIFY_LOG") == "" ]] || fail "top-level used recipient override"
[[ $(wc -l <"$WAKE_LOG") -eq 1 ]] || fail "ata tick wake count: $(cat "$WAKE_LOG")"
grep -qx 'ata commented on epic omp-demo → midi' "$WAKE_LOG" \
  || fail "ata wake line: $(cat "$WAKE_LOG")"

: >"$NOTIFY_LOG"
ata_comment_watch_run || fail "repeat tick failed"
[[ ! -s $NOTIFY_LOG ]] || fail "repeat tick re-notified: $(cat "$NOTIFY_LOG")"
[[ $(wc -l <"$WAKE_LOG") -eq 1 ]] || fail "repeat tick extra wake: $(cat "$WAKE_LOG")"


long=$(printf 'a%.0s' {1..400})
fmt=$(ata_comment_watch_format ata bug "sdo-x" $'line\nbreak "q"' "$long"; printf .)
fmt=${fmt%.}
printf '%s\n' "$fmt" | grep -q "operator ata has commented on your bug sdo-x" \
  || fail "format prefix: $fmt"
printf '%s\n' "$fmt" | grep -q 'line break "q":$' \
  || fail "format title flatten: $fmt"
printf '%s\n' "$fmt" | grep -q "^> ${long}$" \
  || fail "long body was truncated: $fmt"
[[ $fmt == *$'\n\n' ]] || fail "format missing trailing newline"

: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" ata "$long"
ata_comment_watch_run || fail "long tick failed"
got_msg=$(jq -j -r '.msg' "$NOTIFY_LOG"; printf .)
got_msg=${got_msg%.}
printf '%s\n' "$got_msg" | grep -q "^> ${long}$" \
  || fail "live long body truncated"
jq -e '.msg | endswith("\n")' "$NOTIFY_LOG" >/dev/null \
  || fail "live payload missing trailing newline"

parent_id=$(
  add_comment "$BOARD" "omp-demo" midi "the original note"
  last_id "$BOARD"
)
: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" ata $'re:'"$parent_id"$'\nplease look at this full sentence'
ata_comment_watch_run || fail "ata reply tick failed"
got_msg=$(jq -r '.msg' "$NOTIFY_LOG")
printf '%s\n' "$got_msg" | grep -q "^ata replied to your comment on epic omp-demo" \
  || fail "reply header: $got_msg"
printf '%s\n' "$got_msg" | grep -q "midi's comment" \
  && fail "reply named the target: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^midi said:" \
  && fail "addressed midi as midi, want you: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^you said:" \
  || fail "parent said: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^> the original note$" \
  || fail "parent quote: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^ata replied:" \
  || fail "reply said: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^> please look at this full sentence$" \
  || fail "reply quote: $got_msg"
[[ $(jq -r '.recipient' "$NOTIFY_LOG") == midi ]] || fail "reply recipient $(jq -r '.recipient' "$NOTIFY_LOG")"
[[ $(jq -r '.delivery' "$NOTIFY_LOG") == steer ]] || fail "reply not steered"
grep -qx 'ata replied to midi on epic omp-demo' "$WAKE_LOG" \
  || fail "ata reply wake: $(cat "$WAKE_LOG")"

: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" ata $'re:'"$parent_id"$'\nsecond reply on the same parent'
ata_comment_watch_run || fail "chained reply tick failed"
got_msg=$(jq -r '.msg' "$NOTIFY_LOG")
printf '%s\n' "$got_msg" | grep -q "^you said:" \
  || fail "chain you said: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^ata said:" \
  || fail "chain earlier said: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^> please look at this full sentence$" \
  || fail "chain earlier quote: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^ata replied:" \
  || fail "chain latest replied: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^> second reply on the same parent$" \
  || fail "chain latest quote: $got_msg"
n_replied=$(printf '%s\n' "$got_msg" | grep -c "^ata replied:" || true)
[[ $n_replied -eq 1 ]] || fail "chain repeated replied: $got_msg"
[[ $(jq -r '.recipient' "$NOTIFY_LOG") == midi ]] || fail "chain recipient"


zime_id=$(
  add_comment "$BOARD" "omp-demo" zime "zime left this"
  last_id "$BOARD"
)
: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" midi $'re:'"$zime_id"$'\nactor to actor'
ata_comment_watch_run || fail "actor reply tick failed"
got_msg=$(jq -r '.msg' "$NOTIFY_LOG")
printf '%s\n' "$got_msg" | grep -q "^midi replied to your comment" \
  || fail "actor-to-actor header: $got_msg"
printf '%s\n' "$got_msg" | grep -q "zime's comment" \
  && fail "actor-to-actor named the target: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^zime said:" \
  && fail "actor-to-actor named zime, want you: $got_msg"
printf '%s\n' "$got_msg" | grep -q "^you said:" \
  || fail "actor-to-actor you said: $got_msg"
[[ $(jq -r '.recipient' "$NOTIFY_LOG") == zime ]] || fail "actor-to-actor recipient"

grep -qx 'midi replied to zime on epic omp-demo' "$WAKE_LOG" \
  || fail "actor-to-actor wake: $(cat "$WAKE_LOG")"

: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" midi "top-level midi stays silent"
ata_comment_watch_run || fail "midi top-level tick failed"
[[ ! -s $NOTIFY_LOG ]] || fail "top-level non-ata notified: $(cat "$NOTIFY_LOG")"
grep -q 'midi commented' "$WAKE_LOG" && fail "midi top-level recorded a wake"


sf_id=$(
  add_comment "$BOARD" "omp-demo" sf "not an actor"
  last_id "$BOARD"
)
: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" ata $'re:'"$sf_id"$'\nshould skip'
ata_comment_watch_run || fail "sf-parent reply tick failed"
[[ ! -s $NOTIFY_LOG ]] || fail "reply to sf notified: $(cat "$NOTIFY_LOG")"

: >"$NOTIFY_LOG"
add_comment "$BOARD" "omp-demo" ata $'re:99999\nmissing parent'
ata_comment_watch_run || fail "missing parent tick failed"
[[ ! -s $NOTIFY_LOG ]] || fail "missing parent notified: $(cat "$NOTIFY_LOG")"

ORPHAN=$TMP/board-orphan
plant_board "$ORPHAN" ""
add_issue "$ORPHAN" "mg-1" "No lead" task
add_comment "$ORPHAN" "mg-1" ata "should skip"
export ATA_COMMENT_BOARDS=$ORPHAN
: >"$NOTIFY_LOG"
ata_comment_watch_run || fail "orphan seed failed"
add_comment "$ORPHAN" "mg-1" ata "still skip"
ata_comment_watch_run || fail "orphan tick failed"
[[ ! -s $NOTIFY_LOG ]] || fail "leadless board broadcast: $(cat "$NOTIFY_LOG")"

: >"$NOTIFY_LOG"
# shellcheck source=/dev/null
source "$LIB"
[[ ! -s $NOTIFY_LOG ]] || fail "sourcing the lib notified"

BOARD2=$TMP/board-b
plant_board "$BOARD2" hera
add_issue "$BOARD2" "sdo-bn2" "wanted inactive" feature
export ATA_COMMENT_BOARDS=$BOARD2
export STATE_DIR=$TMP/state-b
: >"$NOTIFY_LOG"
"$DRIVER" || fail "driver seed failed"
add_comment "$BOARD2" "sdo-bn2" ata "from the driver"
"$DRIVER" || fail "driver tick failed"
[[ $(jq -s 'length' "$NOTIFY_LOG") -eq 1 ]] || fail "driver notify count: $(cat "$NOTIFY_LOG")"
jq -r '.msg' "$NOTIFY_LOG" | grep -q 'feature sdo-bn2' \
  || fail "driver payload missing type/id: $(cat "$NOTIFY_LOG")"
[[ $(jq -r '.root' "$NOTIFY_LOG") == "$BOARD2" ]] \
  || fail "driver used the wrong scope root"
[[ $(jq -r '.delivery' "$NOTIFY_LOG") == steer ]] \
  || fail "driver did not steer"
grep -qx 'ata commented on feature sdo-bn2 → hera' "$WAKE_LOG" \
  || fail "driver wake: $(cat "$WAKE_LOG")"


echo "comment-watch ok"
