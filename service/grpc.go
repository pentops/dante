package service

import (
	"github.com/pentops/go-grpc-helpers/protovalidatemw"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-auth/o5auth"
	"google.golang.org/grpc"
)

func GRPCMiddleware() []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
		protovalidatemw.UnaryServerInterceptor(),
		o5auth.GRPCMiddleware,
	}
}
