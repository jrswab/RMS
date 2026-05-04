package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"lab37/internal/auth"
	"lab37/internal/db"
	"lab37/internal/handler"
	"lab37/migrations"
)

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	database, err := db.Open(db.DefaultPath())
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	loginHandler := &handler.LoginHandler{DB: database, JWTSecret: []byte(jwtSecret)}
	loginPageHandler := &handler.LoginPageHandler{JWTSecret: []byte(jwtSecret)}
	browserLoginHandler := &handler.BrowserLoginHandler{DB: database, JWTSecret: []byte(jwtSecret)}
	logoutHandler := &handler.LogoutHandler{}
	homeHandler := &handler.HomeHandler{DB: database, JWTSecret: []byte(jwtSecret)}
	searchHandler := &handler.SearchHandler{DB: database}
	recipeViewHandler := &handler.RecipeViewHandler{DB: database}
	recipeCreateHandler := &handler.RecipeCreateHandler{DB: database}
	recipeEditHandler := &handler.RecipeEditHandler{DB: database}
	recipeDeleteHandler := &handler.RecipeDeleteHandler{DB: database}

	r.Get("/login", loginPageHandler.ServeHTTP)
	r.Post("/login", loginHandler.ServeHTTP)
	r.Post("/login/browser", browserLoginHandler.ServeHTTP)
	r.Get("/logout", logoutHandler.ServeHTTP)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware([]byte(jwtSecret)))
		// Future authenticated routes go here
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.BrowserMiddleware([]byte(jwtSecret)))
		r.Post("/search", searchHandler.ServeHTTP)
		r.Get("/recipe/new", recipeCreateHandler.ServeHTTP)
		r.Post("/recipe/new", recipeCreateHandler.ServeHTTP)
		r.Get("/recipe/{id}", recipeViewHandler.ServeHTTP)
		r.Get("/recipe/{id}/edit", recipeEditHandler.ServeHTTP)
		r.Patch("/recipe/{id}", recipeEditHandler.ServeHTTP)
		r.Post("/recipe/{id}/delete-confirm", recipeDeleteHandler.ServeHTTP)
		r.Post("/recipe/{id}/delete-cancel", recipeDeleteHandler.ServeHTTP)
		r.Delete("/recipe/{id}", recipeDeleteHandler.ServeHTTP)
		r.Get("/", homeHandler.ServeHTTP)
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
