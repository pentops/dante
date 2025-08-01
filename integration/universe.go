package integration

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_spb"
	"github.com/pentops/dante/service"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_tpb"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type Universe struct {
	DeadMessageQuery   dante_spb.DeadMessageQueryServiceClient
	DeadMessageWorker  messaging_tpb.DeadMessageTopicClient
	DeadMessageCommand dante_spb.DeadMessageCommandServiceClient
	FakeSqs            FakeSqs

	*flowtest.Stepper[*testing.T]
}

type FakeSqs struct {
	Msgs []sqs.SendMessageInput
}

func (f *FakeSqs) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.Msgs = append(f.Msgs, *params)

	return nil, nil
}

func NewUniverse(ctx context.Context, t *testing.T) *Universe {
	name := t.Name()
	stepper := flowtest.NewStepper[*testing.T](name)
	uu := &Universe{
		Stepper: stepper,
		FakeSqs: FakeSqs{},
	}
	return uu
}

func (uu *Universe) RunSteps(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../ext/db"))
	db := sqrlx.NewPostgres(conn)

	log.DefaultLogger = log.NewCallbackLogger(uu.LevelLog)

	app, err := service.NewApp(db, &uu.FakeSqs, service.Config{})
	if err != nil {
		t.Fatal(err)
	}

	grpcPair := flowtest.NewGRPCPair(t, service.GRPCUnaryMiddleware("dev", false)...)

	app.RegisterGRPC(grpcPair.Server)

	uu.DeadMessageQuery = dante_spb.NewDeadMessageQueryServiceClient(grpcPair.Client)
	uu.DeadMessageCommand = dante_spb.NewDeadMessageCommandServiceClient(grpcPair.Client)
	uu.DeadMessageWorker = messaging_tpb.NewDeadMessageTopicClient(grpcPair.Client)

	grpcPair.ServeUntilDone(t, ctx)

	uu.Stepper.RunSteps(t)
}
