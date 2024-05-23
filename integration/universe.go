package integration

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pentops/dante/dynamictype"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_spb"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_tpb"
	"github.com/pentops/dante/service"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/pgtest.go/pgtest"
)

type Universe struct {
	DeadMessageQuery   dante_spb.DeadMessageQueryServiceClient
	DeadMessageWorker  dante_tpb.DeadMessageTopicClient
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

	log.DefaultLogger = log.NewCallbackLogger(uu.Stepper.Log)

	grpcPair := flowtest.NewGRPCPair(t, service.GRPCMiddleware()...)
	topicPair := flowtest.NewGRPCPair(t, service.GRPCMiddleware()...)

	q, err := service.NewDeadmessagePSM()
	if err != nil {
		t.Fatal(err)
	}

	types := dynamictype.NewTypeRegistry()

	worker, err := service.NewDeadLetterWorker(conn, types, q, "")
	if err != nil {
		t.Fatal(err)
	}

	service, err := service.NewDeadletterServiceService(conn, q, &uu.FakeSqs)
	if err != nil {
		t.Fatal(err)
	}

	dante_spb.RegisterDeadMessageQueryServiceServer(grpcPair.Server, service)
	uu.DeadMessageQuery = dante_spb.NewDeadMessageQueryServiceClient(grpcPair.Client)

	dante_spb.RegisterDeadMessageCommandServiceServer(grpcPair.Server, service)
	uu.DeadMessageCommand = dante_spb.NewDeadMessageCommandServiceClient(grpcPair.Client)

	dante_tpb.RegisterDeadMessageTopicServer(topicPair.Server, worker)
	uu.DeadMessageWorker = dante_tpb.NewDeadMessageTopicClient(topicPair.Client)

	grpcPair.ServeUntilDone(t, ctx)
	topicPair.ServeUntilDone(t, ctx)

	uu.Stepper.RunSteps(t)
}
