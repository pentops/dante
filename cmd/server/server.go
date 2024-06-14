package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	_ "github.com/lib/pq"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_spb"
	"github.com/pentops/dante/service"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_tpb"
	"github.com/pressly/goose"

	"github.com/pentops/envconf.go/envconf"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version string
var sqsClient *sqs.Client

func main() {
	ctx := context.Background()
	ctx = log.WithFields(ctx, map[string]interface{}{
		"application": "dante",
		"version":     Version,
	})

	args := os.Args[1:]
	if len(args) == 0 {
		args = append(args, "serve")
	}

	switch args[0] {
	case "serve":
		if err := runServe(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to serve")
			os.Exit(1)
		}

	case "migrate":
		if err := runMigrate(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to migrate")
			os.Exit(1)
		}

	default:
		log.WithField(ctx, "command", args[0]).Error("Unknown command")
		os.Exit(1)
	}
}

func runMigrate(ctx context.Context) error {
	var config = struct {
		MigrationsDir string `env:"MIGRATIONS_DIR" default:"./ext/db"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := openDatabase(ctx)
	if err != nil {
		return err
	}

	return goose.Up(db, "/migrations")
}

func openDatabase(ctx context.Context) (*sql.DB, error) {
	var config = struct {
		PostgresURL string `env:"POSTGRES_URL"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, err
	}

	// Default is unlimited connections, use a cap to prevent hammering the database if it's the bottleneck.
	// 10 was selected as a conservative number and will likely be revised later.
	db.SetMaxOpenConns(10)

	for {
		if err := db.Ping(); err != nil {
			log.WithError(ctx, err).Error("pinging PG")
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return db, nil
}

func runServe(ctx context.Context) error {
	type envConfig struct {
		PublicPort int    `env:"PUBLIC_PORT" default:"8080"`
		SlackUrl   string `env:"SLACK_URL" default:""`
	}
	cfg := envConfig{}
	if err := envconf.Parse(&cfg); err != nil {
		return err
	}

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	sqsClient = sqs.NewFromConfig(awsConfig)

	db, err := openDatabase(ctx)
	if err != nil {
		return err
	}

	statemachine, err := service.NewDeadmessagePSM()
	if err != nil {
		return err
	}

	deadletterService, err := service.NewDeadletterServiceService(db, statemachine, sqsClient)
	if err != nil {
		return err
	}

	deadletterWorker, err := service.NewDeadLetterWorker(db, statemachine, cfg.SlackUrl)
	if err != nil {
		return err
	}

	log.WithField(ctx, "PORT", cfg.PublicPort).Info("Boot")

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(service.GRPCMiddleware()...))

		reflection.Register(grpcServer)

		messaging_tpb.RegisterDeadMessageTopicServer(grpcServer, deadletterWorker)
		dante_spb.RegisterDeadMessageCommandServiceServer(grpcServer, deadletterService)
		dante_spb.RegisterDeadMessageQueryServiceServer(grpcServer, deadletterService)

		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PublicPort))
		if err != nil {
			return err
		}
		log.WithField(ctx, "port", cfg.PublicPort).Info("Begin Worker Server")
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop() // nolint:errcheck
		}()

		return grpcServer.Serve(lis)
	})

	return eg.Wait()
}
