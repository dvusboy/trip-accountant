## Settlement algorithm for a single expenditure event

Let's start by looking at the individual expenditure event. If there are
N participants at the event, and suppose `p1` and `p2` paid `E1` and `E2`,
respectively. Then each participant should be paying

  ```Math
 (E1 + E2)/N
```

With that, we build the following list:

| p1 | E1  |
| p2 | E2  |
| p3 | 0 |
| p4 | 0 |
| ... | ... |
| pN | 0 |

We then sort the list by the 2nd column so that the person paid out the most
(i.e. the most negative value in the 2nd column) is at the bottom.

Assuming the table above is in the reverse sorted order, we iterate over the list,
starting with `pN`, so from index `pN` to `p1`. At the same time, with a second iterator
going over the list in the opposite direction: from `p1` to `pN`, we look at the payment
the participant made. If it is larger than `2 * (E1 + E2)/N`, we subtract that from
the value, and record `(E1 + E2)/N` to the payment made by the participant on the
first iterator. For example, if `E1 - (E1 + E2)/N > (E1 + E2)/N`, then we update
the value of `p1` to `E1 - (E1 + E2)/N` and change the value for `pN` to `(E1 + E2)/N`.
We record a settlement of the following form:

  ```JSON
{
  "pN>p1" : {
    "payer" : "pN",
    "payee" : "p1",
    "amount" : (E1 + E2)/N
  }
}
```

**NOTE:** The point of the key "pN>p1" is for facilitating the merge of all the
settlements.

Since `pN` has already paid the necessary amount, we advance the first iterator
to `pN-1`. On the other hand, `p1` still paid more than `(E1 + E2)/N`, we do not
advance the second iterator. Then suppose, `E1 - 2 * (E1 + E2)/N < (E1 + E2)/N`,
we can only record a partial settlement, and credit `pN-1` with `E1 - 2 * (E1 + E2)/N`,
while set `p1` to `(E1 + E2)/N`. We record this settlement as:

  ```JSON
{
  "pN-1>p1" : {
    "payer" : "pN-1",
    "payee" : "p1",
    "amount" : E1 - 2 * (E1 + E2)/N
  }
}
```

Since `p1` has reached the settlement share, we advance the second iterator to `p2`.
Meanwhile, `pN-1` is still short of `(E1 + E2)/N`. This means we don't advance the
first iterator. Instead, we look at the difference: `(E1 + E2)/N - (E1 - 2 * (E1 + E2)/N)`,
and credit `p2` with that amount, creating the following settlement entry in the process:

  ```JSON
{
  "pN-1>p2" : {
    "payer" : "pN-1",
    "payee" : "p2",
    "amount" : (E1 + E2)/N - (E1 - 2 * (E1 + E2)/N)
  }
}
```

This will bring the payment from `pN-1` to `(E1 + E2)/N`, while `p2` value will
changed to `E2 - (E1 + E2)/N + E1 - 2 * (E1 + E2)/N`, or `(E1 + E2) - 3 * (E1 + E2)/N`.

We repeat this iteration until all the entries in the table have paid `(E1 + E2)/N`
(see #Considerations below on rounding issue).

## Full settlement record

We repeat the above algorithm for all expenditure events. When adding a new entry
to the settlement record, if the key already exists, we simply add the amounts.
Once all the expenditure events are taken care of, we look for reciprocal settlement
entries in the full record, and merge them. For example, if we have these 2 records:

  ```JSON
  "p1>pN" : {
    "payer" : "p1",
    "payee" : "p2",
    "amount" : A1
  }
```

and,

  ```JSON
  "pN>p1" : {
    "payer" : "pN",
    "payee" : "p1",
    "amount" : A2
  }
```

Then we merge them by picking the entry with the greater `amount`. For example, if
`A1 > A2`, then we just remove the `pN>p1` entry, and just update the `p1>pN` entry:

  ```JSON
  "p1>pN" : {
    "payer" : "p1",
    "payee" : "pN",
    "amount" : A1 - A2
  }
```

If `A1 = A2`, we just eliminate both `p1>pN` and `pN>p1` entries.

### Considerations

The most annoying issue in this algorithm is rounding. Since we are dealing with
amounts in the unit of cent, all payments are integers. But `(E1 + E2)/N` can be
fractional. So, we'll have to round that value since we don't deal with fractional
cent.
