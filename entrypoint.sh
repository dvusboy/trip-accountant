#!/bin/bash

APP=/srv/trip-accountant/bin/trip-accountant
DBFILE=/srv/trip-accountant/data/trips.db

# If --db is in the argument list, extract the DB path
_get_dbpath() {
    local chk_next val opt path scheme

    for opt; do
	if [ -n "$chk_next" ]; then
	    val=$opt
	    break
	fi
	case "$opt" in
	    --db)
		chk_next=1
		;;
	    --db=*)
		val=$(expr "x$opt" : 'x[^=]*=\(.*\)')
		break
		;;
	esac
    done
    if [ -n "$val" ]; then
	# --db option exists and value is in $val
	scheme=$(echo "$val" | awk -F: '{ print $1 }')
	if [ "$scheme" = 'sqlite3' ]; then
	    echo "$val" | sed -e 's|^[^:]\{1,\}://\(.\{1,\}\)$|\1|'
	fi
    fi
}

# If necessary, create the SQLite DB file and then the schema for the app
check_db() {
    local dbpath=$(_get_dbpath "$@")

    if [ -z "$dbpath" ]; then
	dbpath=$DBFILE
    fi

    [ -f "$dbpath" ] || {
	cat <<EOF | sqlite3 "$dbpath"
CREATE TABLE IF NOT EXISTS tuser (
user_id INTEGER CONSTRAINT user_pkey PRIMARY KEY AUTOINCREMENT,
email VARCHAR(256) NOT NULL UNIQUE,
verified BOOLEAN DEFAULT FALSE);

CREATE TABLE IF NOT EXISTS trip (
trip_id INTEGER CONSTRAINT trip_pkey PRIMARY KEY AUTOINCREMENT,
name VARCHAR(128) NOT NULL,
name_lower VARCHAR(128) NOT NULL,
created_at INTEGER NOT NULL,
start_date INTEGER NOT NULL,
end_date INTEGER DEFAULT 0,
description VARCHAR(512));

CREATE TABLE IF NOT EXISTS participant (
trip_id INTEGER NOT NULL,
user_id INTEGER NOT NULL,
is_owner BOOLEAN NOT NULL DEFAULT FALSE,
CONSTRAINT participant_pkey PRIMARY KEY (trip_id, user_id));

CREATE TABLE IF NOT EXISTS expense (
expense_id INTEGER CONSTRAINT expense_pkey PRIMARY KEY AUTOINCREMENT,
trip_id INTEGER NOT NULL,
txn_date INTEGER NOT NULL,
created_at INTEGER NOT NULL,
description VARCHAR(512));
CREATE INDEX IF NOT EXISTS expense_trip_index ON expense(trip_id);

CREATE TABLE IF NOT EXISTS expense_participant (
expense_id INTEGER NOT NULL,
user_id INTEGER NOT NULL,
amount INTEGER NOT NULL,
CONSTRAINT expense_participant_pkey PRIMARY KEY (expense_id, user_id));
EOF
    }
}

# Returns 0 if the file is source as in "source entrypoint.sh"
_is_sourced() {
	[ "${#FUNCNAME[@]}" -ge 2 ] \
	&& [ "${FUNCNAME[0]}" = '_is_sourced' ] \
	&& [ "${FUNCNAME[1]}" = 'source' ]
}

_main() {
	if [ "${1:0:1}" = '-' ]; then
		# It looks like we have options to be passed to APP
		set -- $APP "$@"
	fi

	if [ "$1" = "$APP" ]; then
	    check_db "$@"
	fi
	exec "$@"
}

if ! _is_sourced; then
	_main "$@"
fi
