package server

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"

	"github.com/ThreeDotsLabs/wild-workouts-go-ddd-example/internal/common/auth"
	"github.com/ThreeDotsLabs/wild-workouts-go-ddd-example/internal/common/logs"
)

func RunHTTPServer(createHandler func(router chi.Router) http.Handler) { //createHandler, which is expected to return an http.Handler. This handler will be the main HTTP handler for the application.
	RunHTTPServerOnAddr(":"+os.Getenv("PORT"), createHandler)
}

func RunHTTPServerOnAddr(addr string, createHandler func(router chi.Router) http.Handler) {
	// The reason for using two routers in this scenario is to achieve better organization, separation of concerns, and modularity in handling different parts of the application
	apiRouter := chi.NewRouter()
	setMiddlewares(apiRouter)

	rootRouter := chi.NewRouter()
	// we are mounting all APIs under /api path
	rootRouter.Mount("/api", createHandler(apiRouter))

	logrus.Info("Starting HTTP server")

	err := http.ListenAndServe(addr, rootRouter)
	if err != nil {
		logrus.WithError(err).Panic("Unable to start HTTP server")
	}
}

func setMiddlewares(router *chi.Mux) {
	router.Use(middleware.RequestID) //Request IDs are used for tracking and correlating requests and their corresponding responses in a distributed system.
	// riginal client IP address might get obscured. Instead, the proxy or load balancer's IP address might be set as the RemoteAddr in the request object.
	//* RealIP middleware helps ensure that your application correctly identifies the originating client IP address, even when requests pass through intermediate proxy servers or load balancers.
	router.Use(middleware.RealIP) // This is particularly useful in scenarios where your application is deployed behind a reverse proxy or load balancer.

	// Structured logging is a //? logging approach that records log messages in a structured format, often as key-value pairs.
	router.Use(logs.NewStructuredLogger(logrus.StandardLogger()))
	//^ The Recoverer middleware intercepts panics that occur during the execution of your HTTP handlers.//* It captures the panic,
	//? logs relevant information about the panic (such as the panic message and stack trace), and then safely recovers from the panic,
	//! allowing your server to continue running.
	router.Use(middleware.Recoverer)

	addCorsMiddleware(router)
	addAuthMiddleware(router)

	router.Use(
		middleware.SetHeader("X-Content-Type-Options", "nosniff"), //. The X-Content-Type-Options header is used to prevent browsers from interpreting files as a different MIME type than what is specified by the server. The value "nosniff" indicates that the browser should not perform MIME type sniffing and should strictly adhere to the MIME type specified by the server. This helps prevent certain types of security vulnerabilities, such as MIME confusion attacks.
		middleware.SetHeader("X-Frame-Options", "deny"),           // The X-Frame-Options header is used to control whether a web page can be displayed within an HTML frame or iframe on another website. The value "deny" indicates that the page should not be displayed in a frame or iframe at all. This header helps prevent clickjacking attacks,
	)
	router.Use(middleware.NoCache) //instruct the browser and any intermediary caches not to cache the response
}

func addAuthMiddleware(router *chi.Mux) {
	// If the MOCK_AUTH environment variable is set to true, it uses mock authentication.
	if mockAuth, _ := strconv.ParseBool(os.Getenv("MOCK_AUTH")); mockAuth {
		router.Use(auth.HttpMockMiddleware)
		return
	}

	var opts []option.ClientOption
	if file := os.Getenv("SERVICE_ACCOUNT_FILE"); file != "" {
		opts = append(opts, option.WithCredentialsFile(file))
	}

	config := &firebase.Config{ProjectID: os.Getenv("GCP_PROJECT")}
	// !The code initializes a Firebase app and authentication client for every incoming request
	firebaseApp, err := firebase.NewApp(context.Background(), config, opts...) //client option
	if err != nil {
		logrus.Fatalf("error initializing app: %v\n", err)
	}

	authClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		// The Recoverer middleware will capture this panic and log information about it, but it won't be able to prevent the application from terminating, because the Fatal log level explicitly instructs the application to exit.
		// Instead of using logrus.Fatalf in your addAuthMiddleware function, you might consider using logrus.Errorf to log the error without terminating the application.
		logrus.WithError(err).Fatal("Unable to create firebase Auth client")
	}

	router.Use(auth.FirebaseHttpMiddleware{authClient}.Middleware)
}

func addCorsMiddleware(router *chi.Mux) {
	allowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ";")
	if len(allowedOrigins) == 0 {
		return
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"}, // The list of headers that the server may expose in the response.
		AllowCredentials: true,             //cookies
		MaxAge:           300,              //: The maximum time (in seconds) that preflight responses can be cached.
	})
	router.Use(corsMiddleware.Handler)
}
