package madhatter

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"context"

	"github.com/stretchr/testify/assert"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(h Handler) Handler {
		return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			existingTag, _ := ctx.Value("tag").(string)
			ctx = context.WithValue(ctx, "tag", existingTag+tag)

			h.ServeHTTP(ctx, w, r)
		})
	}
}

// Not recommended (https://golang.org/pkg/reflect/#Value.Pointer),
// but the best we can do.
func funcsEqual(f1, f2 interface{}) func() bool {
	return func() bool {
		val1 := reflect.ValueOf(f1)
		val2 := reflect.ValueOf(f2)
		return val1.Pointer() == val2.Pointer()
	}
}

var testApp = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	existingTag, _ := ctx.Value("tag").(string)
	w.Write([]byte(existingTag + "app\n"))
})

// Tests creating a new chain
func TestNew(t *testing.T) {
	c1 := func(h Handler) Handler {
		return nil
	}

	c2 := func(h Handler) Handler {
		return nil
	}

	slice := []Constructor{c1, c2}
	chain := New(slice...)
	assert.Condition(t, funcsEqual(chain.constructors[0], slice[0]))
	assert.Condition(t, funcsEqual(chain.constructors[1], slice[1]))
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	assert.NotPanics(t, func() {
		chain := New()
		final := chain.Then(testApp)

		assert.NotNil(t, final)

		w := httptest.NewRecorder()
		final.ServeHTTP(w, nil)

		assert.Equal(t, w.Body.String(), "app\n")
	})
}

func TestThenTreatsNilAsDefaultServeMux(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained := New().Then(nil)
	chained.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestThenFuncTreatsNilAsDefaultServeMux(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained := New().ThenFunc(nil)
	chained.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestThenOrdersHandlersRight(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained := New(t1, t2, t3).Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String(), "t1\nt2\nt3\napp\n")
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))

	assert.Equal(t, len(chain.constructors), 2)
	assert.Equal(t, len(newChain.constructors), 4)

	chained := newChain.ThenFunc(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String(), "t1\nt2\nt3\nt4\napp\n")
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Append(tagMiddleware(""))

	assert.NotEqual(t, &chain.constructors[0], &newChain.constructors[0])
}

func TestNegroniAdapter(t *testing.T) {
	n1 := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		w.Write([]byte("t1\n"))
		next(w, r)
	}

	chain := New(AdaptInstance(NegroniHandlerFunc(n1)))
	newChain := chain.Append(tagMiddleware("t2\n"))
	chained := newChain.ThenFunc(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String(), "t1\nt2\napp\n")
}
