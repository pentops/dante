package integration

import (
	"context"
	"testing"

	"github.com/pentops/dante/dynamictype"
	"github.com/pentops/dante/service"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/dante/v1/dante_spb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"
	"github.com/pentops/pgtest.go/pgtest"
)

type Universe struct {
	DeadMessageQuery   dante_spb.DeadMessageQueryServiceClient
	DeadMessageWorker  dante_tpb.DeadMessageTopicClient
	DeadMessageCommand dante_spb.DeadMessageCommandServiceClient

	*flowtest.Stepper[*testing.T]
}

func NewUniverse(ctx context.Context, t *testing.T) *Universe {
	name := t.Name()
	stepper := flowtest.NewStepper[*testing.T](name)
	uu := &Universe{
		Stepper: stepper,
	}
	return uu
}

func (uu *Universe) RunSteps(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../ext/db"))

	log.DefaultLogger = log.NewCallbackLogger(uu.Stepper.Log)

	grpcPair := flowtest.NewGRPCPair(t)
	topicPair := flowtest.NewGRPCPair(t)

	types := dynamictype.NewTypeRegistry()
	service, err := service.NewDeadletterServiceService(conn, types, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	dante_spb.RegisterDeadMessageQueryServiceServer(grpcPair.Server, service)
	uu.DeadMessageQuery = dante_spb.NewDeadMessageQueryServiceClient(grpcPair.Client)

	dante_spb.RegisterDeadMessageCommandServiceServer(grpcPair.Server, service)
	uu.DeadMessageCommand = dante_spb.NewDeadMessageCommandServiceClient(grpcPair.Client)

	dante_tpb.RegisterDeadMessageTopicServer(topicPair.Server, service)
	uu.DeadMessageWorker = dante_tpb.NewDeadMessageTopicClient(topicPair.Client)

	grpcPair.ServeUntilDone(t, ctx)
	topicPair.ServeUntilDone(t, ctx)

	uu.Stepper.RunSteps(t)
}
