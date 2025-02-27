## Part 2: Data Modeling

I'll be using a relational DB. This choice is mostly because I'm most
familiar with it.

#### TUser:

| Column Name | Data Type | Constraints |
| --- | --- | --- |
| user_id | integer | not null, primary key (from sequence) |
| email | varchar(256) | not null, unique |
| verified | boolean | default false |

In SQL:

  ```SQL
CREATE SEQUENCE user_id_seq;
CREATE TABLE tuser (
  user_id INTEGER DEFAULT nextval('user_id_seq') CONSTRAINT user_pkey PRIMARY KEY
  , email VARCHAR(256) NOT NULL
  , verified BOOLEAN DEFAULT false
);
CREATE UNIQUE INDEX tuser_email_index ON tuser (email);
```

#### Trip:

| Column Name | Data Type | Constraints |
| --- | --- | --- |
| trip_id | integer | not null, primary key (from sequence) |
| name | varchar(128) | not null |
| name_lower | varchar(128) | not null (lowercase of "name") |
| created_at | integer | not null (Epoch timestamp in µs) |
| start_date | integer | not null (Epoch timestamp) |
| end_date | integer | default 0 (Epoch timestamp) |
| description | varchar(512) | |

In SQL:

  ```SQL
CREATE SEQUENCE trip_id_seq;
CREATE TABLE trip (
  trip_id INTEGER CONSTRAINT trip_pkey PRIMARY KEY
  , name VARCHAR(128) NOT NULL
  , name_lower VARCHAR(128) NOT NULL
  , created_at INTEGER NOT NULL
  , start_date INTEGER NOT NULL
  , end_date INTEGER DEFAULT 0
  , description VARCHAR(512)
);
CREATE INDEX trip_name_index ON trip (name_lower);
```

#### Participant:

| Column Name | Data Type | Constraints |
| --- | --- | --- |
| trip_id | integer | not null, foreign key "trip.trip_id", compound primary key with "user_id" |
| user_id | integer | not null, foreign key "tuser.user_id", compound primary key with "trip_id" |
| is_owner | boolean | not null, default false |

In SQL:

  ```SQL
CREATE TABLE participant (
  trip_id INTEGER NOT NULL
  , user_id INTEGER NOT NULL
  , is_owner BOOLEAN NOT NULL DEFAULT false
  , CONSTRAINT participant_pkey PRIMARY KEY (trip_id, user_id)
);
```

#### Expense:

| Column Name | Data Type | Constraints |
| --- | --- | --- |
| expense_id | integer | not null, primary key (from sequence) |
| trip_id | integer | not null, foreign key "trip.trip_id" |
| txn_date | integer | not null (Epoch timestamp) |
| created_at | integer | not null (Epoch timestamp in µs) |
| DESCRIPTION | varchar(512) | |

**NOTE:**

* add "currency_id" to support multiple currencies

In SQL:

  ```SQL
CREATE SEQUENCE expense_id_seq;
CREATE TABLE expense (
  expense_id INTEGER CONSTRAINT expense_pkey PRIMARY KEY
  , trip_id INTEGER NOT NULL
  , txn_date INTEGER NOT NULL
  , created_at INTEGER NOT NULL
  , description VARCHAR(512)
);
CREATE INDEX expense_trip_index ON expense (trip_id);
```

#### Expense_Participant:

| Column Name | Data Type | Constraints |
| --- | --- | --- |
| expense_id | integer | not null, foreign key "expense.expense_id", compound primary key with "user_id" |
| user_id | integer | not null, foreign key "tuser.user_id", compound primary key with "expense_id" |
| amount | integer | not null (in cent) |

In SQL:

  ```SQL
CREATE TABLE expense_participant (
  expense_id INTEGER NOT NULL
  , user_id INTEGER NOT NULL
  , amount INTEGER NOT NULL
  , CONSTRAINT expense_participant_pkey PRIMARY KEY (expense_id, user_id)
);
```

**NOTE:**

We need to ensure that there is at least one row for a given "expense_id" that has "amount" greater than 0.
That means the following query should return a value greater than 0.

  ```SQL
SELECT SUM(amount) FROM expense_participant WHERE expense_id=<given ID>;
```

And the total expenditure of a trip would be the value of this query:

  ```SQL
SELECT SUM(ep.amount) FROM
  expense_participant AS ep
  , expense AS e
WHERE e.trip_id=<give ID>
  AND e.expense_id=ep.expense_id;
```

Whereas, the total expenditure of a participant during a trip is:

  ```SQL
SELECT u.email AS participant
  , SUM(ep.amount) AS expense
FROM expense_participant AS ep
  , expense AS e
  , tuser AS u
WHERE e.trip_id=<given ID>
  AND e.expense_id=ep.expense_id
  AND ep.user_id=u.user_id
GROUP BY ep.user_id;
```

#### Trip_Settlement

| Column Name | Data Type | Constraints |
| --- | --- | --- |
| trip_id | INTEGER | not null, foreign key "trip.trip_id" |
| payer | INTEGER | not null, foreign key "tuser.user_id" |
| payee | INTEGER | not null, foreign key "tuser.user_id" |
| amount | INTEGER | not null (in cent) |

** NOTE: **

Obviously, "payer" != "payee", should be handled in code.

In SQL:

  ```SQL
CREATE TABLE trip_settlement (
  trip_id INTEGER NOT NULL
  , payer INTEGER NOT NULL
  , payee INTEGER NOT NULL
  , amount INTEGER NOT NULL
  CONSTRAINT trip_settlement_pkey PRIMARY KEY (trip_id, payer, payee)
);
```
