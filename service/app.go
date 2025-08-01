package service

import (
	"github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
	"github.com/pentops/grpc.go/protovalidatemw"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/realms/j5auth"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc"
)

type App struct {
	deadletterService *DeadletterService
	deadletterWorker  *DeadLetterWorker
	statemachine      *dante_pb.DeadmessagePSM
}

type Config struct {
	SlackUrl string `env:"SLACK_URL" default:""`
}

func NewApp(db sqrlx.Transactor, sqsClient SqsSender, cfg Config) (*App, error) {

	statemachine, err := NewDeadmessagePSM()
	if err != nil {
		return nil, err
	}

	deadletterService, err := NewDeadletterServiceService(db, statemachine, sqsClient)
	if err != nil {
		return nil, err
	}

	deadletterWorker, err := NewDeadLetterWorker(db, statemachine, cfg.SlackUrl)
	if err != nil {
		return nil, err
	}

	return &App{
		deadletterService: deadletterService,
		deadletterWorker:  deadletterWorker,
		statemachine:      statemachine,
	}, nil
}

func (a *App) RegisterGRPC(s grpc.ServiceRegistrar) {
	a.deadletterService.RegisterGRPC(s)
	a.deadletterWorker.RegisterGRPC(s)
}

func GRPCUnaryMiddleware(version string, validateReply bool) []grpc.UnaryServerInterceptor {
	var pvOpts []protovalidatemw.Option
	if validateReply {
		pvOpts = append(pvOpts, protovalidatemw.WithReply())
	}
	return []grpc.UnaryServerInterceptor{
		grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
		j5auth.GRPCMiddleware,
		protovalidatemw.UnaryServerInterceptor(pvOpts...),
	}
}
