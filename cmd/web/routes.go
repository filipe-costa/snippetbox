package main

import (
	"net/http"

	"github.com/justinas/alice"
	"snippetbox.fcosta.dev/ui"
)

// The routes() method returns a servemux containing our application routes.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(ui.Files))
	mux.HandleFunc("GET /ping", ping)

	// Create a new middleware chain containing the middleware specific to our
	// dynamic application routes. For now, this chain will only contain the
	// LoadAndSave session middleware but we'll add more to it later.
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	// Add the five new routes, all of which use our 'dynamic' middleware chain.
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /about", dynamic.ThenFunc(app.about))
	mux.Handle("GET /snippet/view/{id}", dynamic.ThenFunc(app.snippetView))
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))

	// Protected (authenticated-only) application routes, using a new "protected"
	// middleware chain which includes the requireAuthentication middleware.
	protected := dynamic.Append(app.requireAuthentication)

	mux.Handle("GET /snippet/create", protected.ThenFunc(app.snippetCreate))
	mux.Handle("POST /snippet/create", protected.ThenFunc(app.snippetCreatePost))
	// Add the view account route, using the protected middleware chain.
	mux.Handle("GET /account/view", protected.ThenFunc(app.accountView))
	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))

	mux.Handle("GET /user/password/update", protected.ThenFunc(app.accountPasswordUpdate))
	mux.Handle("POST /user/password/update", protected.ThenFunc(app.accountPasswordUpdatePost))

	// Create a middleware chain containing our 'standard' middleware
	// which will be used for every request our application receives.
	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	// Return the 'standard' middleware chain followed by the servemux.
	return standard.Then(mux)
}
