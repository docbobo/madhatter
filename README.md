Mad Hatter
==========

[![Build Status](https://travis-ci.org/docbobo/madhatter.svg?branch=master)](https://travis-ci.org/docbobo/madhatter)

Mad Hatter provides a convenient way to chain HTTP middleware functions, similar to [Alice](https://github.com/justinas/alice).
It just differs by the fact that it does not expect middleware to fulfill the http.Handler interface - instead it comes with
it's own Handler interface that passes a context.Context in addition.

