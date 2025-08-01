package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	_ "github.com/lib/pq"
	"github.com/pentops/dante/service"
	"github.com/pentops/grpc.go/grpcbind"
	"github.com/pentops/j5/lib/psm/psmigrate"
	"github.com/pentops/runner/commander"
	"github.com/pentops/sqrlx.go/pgenv"
	"github.com/pressly/goose"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version string
var sqsClient *sqs.Client

func main() {
	mainGroup := commander.NewCommandSet()

	mainGroup.Add("serve", commander.NewCommand(runServe))
	mainGroup.Add("migrate", commander.NewCommand(runMigrate))
	mainGroup.Add("psm-tables", commander.NewCommand(runPSMTables))
	mainGroup.RunMain("dante", Version)
}

func runPSMTables(ctx context.Context, cfg struct{}) error {
	stateMachine, err := service.NewDeadmessagePSM()
	if err != nil {
		return fmt.Errorf("failed to create state machine: %w", err)
	}
	specs := stateMachine.StateTableSpec()
	migrationFile, err := psmigrate.BuildStateMachineMigrations(specs)
	if err != nil {
		return fmt.Errorf("build migration file: %w", err)
	}
	fmt.Println(string(migrationFile))
	return nil
}

func runMigrate(ctx context.Context, cfg struct {
	MigrationsDir string `env:"MIGRATIONS_DIR" default:"./ext/db"`
	pgenv.DatabaseConfig
}) error {

	db, err := cfg.OpenPostgres(ctx)
	if err != nil {
		return err
	}

	return goose.Up(db, cfg.MigrationsDir)
}

func runServe(ctx context.Context, cfg struct {
	grpcbind.EnvConfig
	pgenv.DatabaseConfig
	service.Config
}) error {
	db, err := cfg.OpenPostgresTransactor(ctx)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		service.GRPCUnaryMiddleware(Version, false)...,
	))

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	sqsClient = sqs.NewFromConfig(awsConfig)

	appSet, err := service.NewApp(db, sqsClient, cfg.Config)
	if err != nil {
		return fmt.Errorf("failed to build portfolio: %w", err)
	}
	appSet.RegisterGRPC(grpcServer)
	reflection.Register(grpcServer)

	return cfg.ListenAndServe(ctx, grpcServer)
}
