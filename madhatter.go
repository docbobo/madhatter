package madhatter

import (
	"net/http"

	"golang.org/x/net/context"
)

// Handler provides an interface similar to http.Handler but supporting
// pass-through of context
type Handler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions
// as HTTP handlers supporting context.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// ServeHTTP calls f(ctx, w, r).
func (f HandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

// A Constructor for a piece of middleware.
type Constructor func(Handler) Handler

// Chain acts as list of Handlers.
// Chain is effectively immutable: once created, it will always hold
// the same set of constructors in the same order.
type Chain struct {
	constructors []Constructor

	finalize func(Handler) http.Handler
}

// New creates a new Chain, memorizing the given list of middleware constructors.
// Constructors will not be called until calling Then() or ThenFunc()
func New(constructors ...Constructor) Chain {
	c := Chain{
		finalize: createRootHandler,
	}
	c.constructors = append(c.constructors, constructors...)
	return c
}

// Then chains the middleware and returns the final http.Handler (!)
//     New(m1, m2, m3).Then(h)
// is equivalent to:
//     m1(m2(m3(h)))
//
// A Chain can be safely reused by calling Then() several times.
// Note that constructors will be called on every call to Then() and thus
// several instances of the same middleware will be created when a Chain
// is reused.
//
// Then() treats nil as http.DefaultServeMux.
func (c Chain) Then(h Handler) http.Handler {
	var final Handler
	if h != nil {
		final = h
	} else {
		final = adaptFinal(http.DefaultServeMux)
	}

	for i := len(c.constructors) - 1; i >= 0; i-- {
		final = c.constructors[i](final)
	}

	return c.finalize(final)
}

// ThenFunc works identically to Then(), but takes a HandlerFunc instead of
// a Handler.
//
// The following to statements are equivalent:
//     c.Then(HandlerFunc(fn))
//     c.ThenFunc(fn)
func (c Chain) ThenFunc(fn HandlerFunc) http.Handler {
	if fn == nil {
		return c.Then(nil)
	}

	return c.Then(HandlerFunc(fn))
}

// Append extends a Chain, adding the specified constructors at the end of the Chain.
// Append returns a new chain, leaving the original one untouched.
func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, len(c.constructors)+len(constructors))
	copy(newCons, c.constructors)
	copy(newCons[len(c.constructors):], constructors)

	newChain := New(newCons...)
	return newChain
}

func createRootHandler(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx    context.Context
			cancel context.CancelFunc
		)

		ctx, cancel = context.WithCancel(context.Background())
		defer cancel() // cancel context as soon as the request returns

		h.ServeHTTP(ctx, w, r)
	})
}

func adaptFinal(h http.Handler) Handler {
	return HandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}
