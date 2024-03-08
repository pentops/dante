package integration

import (
	"context"
	"testing"

	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/dante/v1/dante_spb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"
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

	//conn := pgtest.GetTestDB(t, pgtest.WithDir("../ext/db"))

	log.DefaultLogger = log.NewCallbackLogger(uu.Stepper.Log)

	grpcPair := flowtest.NewGRPCPair(t)
	topicPair := flowtest.NewGRPCPair(t)

	/*
		sm, err := states.NewStateMachine()
		if err != nil {
			t.Fatal(err)
		}

		queryService, err := service.NewDeadMessageQueryService(conn, sm)
		if err != nil {
			t.Fatal(err)
		}

		dante_spb.RegisterDeadMessageQueryServiceServer(grpcPair.Server, queryService)
		uu.DeadMessageQuery = dante_spb.NewDeadMessageQueryServiceClient(grpcPair.Client)

		commandService, err := service.NewDeadMessageCommandService(conn, sm)
		if err != nil {
			t.Fatal(err)
		}

		dante_spb.RegisterDeadMessageCommandServiceServer(grpcPair.Server, commandService)
		uu.DeadMessageCommand = dante_spb.NewDeadMessageCommandServiceClient(grpcPair.Client)

		deadMessageWorker, err := worker.NewDeadMessageWorker(conn)
		if err != nil {
			t.Fatal(err)
		}

		dante_tpb.RegisterDeadMessageTopicServer(topicPair.Server, deadMessageWorker)
		uu.DeadMessageWorker = dante_tpb.NewDeadMessageTopicClient(topicPair.Client)
	*/

	grpcPair.ServeUntilDone(t, ctx)
	topicPair.ServeUntilDone(t, ctx)

	uu.Stepper.RunSteps(t)
}
