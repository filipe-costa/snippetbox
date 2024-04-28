package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	// New import
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
	"snippetbox.fcosta.dev/internal/models"
)

type application struct {
	debug          bool
	logger         *slog.Logger
	snippets       models.SnippetModelInterface
	users          models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	// Define a new command-line flag for the MySQL DSN string.
	dsn := flag.String("dsn", "web:pass@tcp(localhost:3306)/snippetbox?parseTime=true", "MySQL data source name")
	debug := flag.Bool("debug", false, "Enable Debug mode")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// To keep the main() function tidy I've put the code for creating a connection
	// pool into the separate openDB() function below. We pass openDB() the DSN
	// from the command-line flag.
	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	// We also defer a call to db.Close(), so that the connection pool is closed
	// before the main() function exits.
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Use the scs.New() function to initialize a new session manager. Then we
	// configure it to use our MySQL database as the session store, and set a
	// lifetime of 12 hours (so that sessions automatically expire 12 hours
	// after first being created).
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	// Make sure that the Secure attribute is set on our session cookies.
	// Setting this means that the cookie will only be sent by a user's web
	// browser when a HTTPS connection is being used (and won't be sent over an
	// unsecure HTTP connection).
	sessionManager.Cookie.Secure = true

	formDecoder := form.NewDecoder()

	app := &application{
		debug:          *debug,
		logger:         logger,
		snippets:       &models.SnippetModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}
	// Initialize a new http.Server struct. We set the Addr and Handler fields so
	// that the server uses the same network address and routes as before.
	srv := &http.Server{
		Addr:           *addr,
		Handler:        app.routes(),
		MaxHeaderBytes: 524288,
		// Create a *log.Logger from our structured logger handler, which writes
		// log entries at Error level, and assign it to the ErrorLog field. If
		// you would prefer to log the server errors at Warn level instead, you
		// could pass slog.LevelWarn as the final parameter.
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
		// Add Idle, Read and Write timeouts to the server.
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.Info("starting server", "addr", srv.Addr)
	// Call the ListenAndServe() method on our new http.Server struct to start
	// the server.
	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

// The openDB() function wraps sql.Open() and returns a sql.DB connection pool
// for a given DSN.
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
