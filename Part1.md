## Part 1: Understanding the problem

The initial problem can be summarized as follow:

* A trip contains a list of participants. A "owner" creates the trip object
and invites the other participants.
* A trip also contains a list of expenses.
* For each expense, there are:
  * a date of the expense
  * a description of the event leading to the expense
  * one or more payers of the expense
  * other participants of the event (this can be a subset of the group of participants of the trip)
* At the end of the trip, for each participant:
  * compute a list of payees (which can be empty), and
  * for each payee, how much money the participant should pay

For simplicity of the MVP, we would assume:

* Email addresses are the unique identifiers for users
* We are not handling authentication or the verification of any email addresses
* We are not handling notification
* Non-participating payers are not supported
* Different pricing for different participant is not supported (e.g. adult vs children tickets)

Further enhancements:

* Support authentication
* Support email validation
* Support integration with payment vendors like PayPal, Venmo, etc.
* The trip can be out of US, with a different currency for the expenses
  * this could mean we don't know the exact transaction amount until it is posted to the credit card issuer
* It can also be the group of participants have credit cards issued in different base currencies.
This can get complicated fast.
* Support varying price-point for an event
* Support non-equal distribution of the total expense, e.g. we have multiple families participating in the trip
  * Families may have younger kids that don't cost as much (such as children tickets, or kid's entrée)
