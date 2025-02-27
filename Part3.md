## Part 3: API Design

### Create a trip

The front-end will host a form with the following fields:

  * owner's email address
  * a short-name of the trip
  * a start date in the form of YYYY-MM-DD
  * a description of the trip
  * a list of email addresses of other participants

The form is to `POST` to the following URL:

  http://localhost/trips

with these parameters, expressed as a JSON document:

  ```JSON
{
	"owner" : "<email address of the owner of the trip>",
	"name"  : "<trip name>",
	"start_date" : "YYYY-MM-DD",
	"description" : "<some longer description of the trip>",
	"participants" : [
		"<email address>",
		...
	],
}
```

Behind the scene, for each email address provided if it
isn't in the list of registered user, a verification email
message should be sent, and a new user record should also
be created.

#### Error conditions

If there are duplicate email addresses in the list of participants,
only the unique list will be used. If the list of participants
also contains the owner, it will be ignored.

`400 Bad Request`:
 * if any email address is invalid
 * if the start date is invalid

#### Returned value

`201 Created`

  ```JSON
{
	"trip_id" : <ID of the trip>
}
```

`200 OK`

If the trip already exist (with the same owner, normalized
short-name (in lowercase), and start date), we consider the
request as an update, especially on the list of participants.

  ```JSON
{
	"trip_id" : <same ID>
}
```

### List active trips

The following URL, basically a `GET`, should provide a list of trips for
a given owner email address, and do not have a non-zero value for `end_date`,
thus still active.

  http://localhost/<owner email>/trips

#### Returned value

`200 OK`:

  ```JSON
{
	"<short name of the trip>" : {
		"trip_id" : <ID>,
		"owner" : {
			"user_id" : <ID>,
			"email" : "<email address>",
			"verified" : <boolean>
		},
		"name" : "<short name of the trip>",
		"start_date" : "YYYY-MM-DD",
		"end_date" : "",
		"description" : "<longer description>",
		"participants" : [
			{
				"user_id" : <ID>,
				"email" : "<email address>",
				"verified" : <boolean>
			},
			...
		]
	},
	...
}
```

### Add expense to a trip

This is performed with a `POST` to the following URL:

  http://localhost/trips/<trip ID>/expenses

with a JSON payload like this:

  ```JSON
{
	"date" : "YYYY-MM-DD",
	"description" : "...",
	"participants" : {
		"<email address>" : <amount paid in cent>,
		...
	}
}
```

Here we are assuming single payer for the whole expense transaction.

#### Error conditions

In the case there are duplicate email address in the list of participants,
it'll be handle the same as the creation of the trip: duplicates would be
treated as a single instance.

`400 Bad Request`:
  * if there are invalid email addresses
  * insensible date

`404 Not Found`:
  * invalid trip ID

#### Returned value

`202 Accepted`

  ```JSON
{
	"expense_id" : <ID>
}
```

### List all expenses for a given trip

The same URI as posting expenses is used to list all the expenses

  http://localhost/trips/<trip ID>/expenses

via a `GET` operation.

#### Returned value

A list of expense object,

  ```JSON
[
	{
		"expense_id" : <ID>,
		"date" : "YYYY-MM-DD",
		"description" : "...",
		"participants" : [
			{
				"email" : "<email address>",
				"paid" : <amount paid in cent>
			},
			...
		]
	},
	...
]
```

### Get the settlement

  http://localhost/trips/<trip ID>/settlement

#### Returned value

  ```JSON
{
	"<email address of payer1>" : {
		"<email address of payee1>" : <amount to payee1 in cent>,
		"<email address of payee2>" : <amount to payee2 in cent>,
		...
	},
	"<email address of payer2>" : {
		"<email address of payee1>" : <amount to payee1 in cent>,
		...
	},
	...
}
```

#### Error conditions

`404 Not Found`:
  * invalid trip ID
