package main

import (
	"context"
	"crypto/rsa"
	"expvar" // Register the expvar handlers
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof" // Register the pprof handlers
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/ardanlabs/service/app/sales-api/handlers"
	"github.com/ardanlabs/service/business/auth"
	"github.com/ardanlabs/service/foundation/database"
	"github.com/ardanlabs/service/foundation/messaging"
	"github.com/ardanlabs/service/foundation/tracer"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

/*
ZipKin: http://localhost:9411
AddLoad: hey -m GET -c 10 -n 10000 "http://localhost:3000/v1/users"
expvarmon -ports=":4000" -vars="build,requests,goroutines,errors,mem:memstats.Alloc"
*/

/*
Need to figure out timeouts for http service.
You might want to reset your DB_HOST env var during test tear down.
Service should start even without a DB running yet.
symbols in profiles: https://github.com/golang/go/issues/23376 / https://github.com/google/pprof/pull/366
*/

// build is the git version of this program. It is set using build flags in the makefile.
var build = "develop"

func main() {
	log := log.New(os.Stdout, "SALES : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(log); err != nil {
		log.Println("main: error:", err)
		os.Exit(1)
	}
}

func run(log *log.Logger) error {

	// =========================================================================
	// Configuration

	var cfg struct {
		conf.Version
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3000"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s"`
		}
		DB struct {
			User       string `conf:"default:postgres"`
			Password   string `conf:"default:postgres,noprint"`
			Host       string `conf:"default:db"`
			Name       string `conf:"default:postgres"`
			DisableTLS bool   `conf:"default:false"`
		}
		KAFKA struct {
			URL string `conf:"default:kafka:9092"`
			// BootstrapServers string `conf:"default:localhost:9092"`
			TOPIC string `conf:"default:product"`
		}
		Auth struct {
			KeyID          string `conf:"default:54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"`
			PrivateKeyFile string `conf:"default:/app/private.pem"`
			Algorithm      string `conf:"default:RS256"`
		}
		Zipkin struct {
			ReporterURI string  `conf:"default:http://zipkin:9411/api/v2/spans"`
			ServiceName string  `conf:"default:sales-api"`
			Probability float64 `conf:"default:0.05"`
		}
	}
	cfg.Version.SVN = build
	cfg.Version.Desc = "copyright information here"

	if err := conf.Parse(os.Args[1:], "SALES", &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage("SALES", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString("SALES", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	// =========================================================================
	// KAFKA
	kafkaWriter := messaging.GetKafkaWriter(cfg.KAFKA.URL, cfg.KAFKA.TOPIC, log)
	defer kafkaWriter.Close()

	// err := kafkaWriter.WriteMessages(context.Background(), kafka.Message{Key: []byte("rahul33333"), Value: []byte("more")})
	// if err != nil {
	// 	return err
	// }

	// =========================================================================
	// App Starting

	// Print the build version for our logs. Also expose it under /debug/vars.
	expvar.NewString("build").Set(build)
	log.Printf("main : Started : Application initializing : version %q", build)
	defer log.Println("main: Completed")

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main: Config :\n%v\n", out)

	// =========================================================================
	// Initialize authentication support

	log.Println("main : Started : Initializing authentication support")

	privatePEM, err := ioutil.ReadFile(cfg.Auth.PrivateKeyFile)
	if err != nil {
		return errors.Wrap(err, "reading auth private key")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return errors.Wrap(err, "parsing auth private key")
	}

	keyLookupFunc := func(publicKID string) (*rsa.PublicKey, error) {
		switch publicKID {
		case cfg.Auth.KeyID:
			return privateKey.Public().(*rsa.PublicKey), nil
		}
		return nil, fmt.Errorf("no public key found for the specified kid: %s", publicKID)
	}
	a, err := auth.New(privateKey, cfg.Auth.KeyID, cfg.Auth.Algorithm, keyLookupFunc)
	if err != nil {
		return errors.Wrap(err, "constructing auth")
	}

	// =========================================================================
	// Start Database

	log.Println("main: Initializing database support")

	db, err := database.Open(database.Config{
		User:       cfg.DB.User,
		Password:   cfg.DB.Password,
		Host:       cfg.DB.Host,
		Name:       cfg.DB.Name,
		DisableTLS: cfg.DB.DisableTLS,
	})
	if err != nil {
		return errors.Wrap(err, "connecting to db")
	}
	defer func() {
		log.Printf("main: Database Stopping : %s", cfg.DB.Host)
		db.Close()
	}()

	// =========================================================================
	// Start Tracing Support

	// WARNING: The current Init settings are using defaults which I listed out
	// for readability. Please review the documentation for opentelemetry.

	log.Println("main: Initializing zipkin tracing support")

	if err := tracer.Init(cfg.Zipkin.ServiceName, cfg.Zipkin.ReporterURI, cfg.Zipkin.Probability, log); err != nil {
		return errors.Wrap(err, "starting tracer")
	}

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.

	log.Println("main: Initializing debugging support")

	go func() {
		log.Printf("main: Debug Listening %s", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, http.DefaultServeMux); err != nil {
			log.Printf("main: Debug Listener closed : %v", err)
		}
	}()

	// =========================================================================
	// Start API Service

	log.Println("main: Initializing API support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handlers.API(build, shutdown, log, db, a),
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Printf("main: API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Printf("main: %v : Start shutdown", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and shed load.
		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return errors.Wrap(err, "could not stop server gracefully")
		}
	}

	return nil
}
