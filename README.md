The goal of this project is to build a backend service that helps
tracking expenses during a trip and at the end computes the settlement
to square off the money efficiently.

The discussion has been broken down into 4 parts:

  * [Part 1](Part1.md) discusses the problem at hand and the limitations of
  the current implementation
  * [Part 2](Part2.md) presents the data modeling and schema design
  * [Part 3](Part3.md) is the API outline
  * [Part 4](Part4.md) talks about the algorithm used in the settlement of
  the expenses

### Limitations

* There is no edit, such as changing participants to a trip or an
expenditure event. Not even changing a user's email address.

