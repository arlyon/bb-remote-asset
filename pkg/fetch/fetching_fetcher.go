package fetch

import (
	"context"
	"io"

	remoteasset "github.com/bazelbuild/remote-apis/build/bazel/remote/asset/v1"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildbarn/bb-remote-asset/pkg/qualifier"
	"github.com/buildbarn/bb-remote-asset/pkg/translator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fetchingFetcher struct {
	fetcher                         Fetcher
	pusher                          remoteasset.PushServer
	actionCacheClient               remoteexecution.ActionCacheClient
	contentAddressableStorageClient remoteexecution.ContentAddressableStorageClient
	executionClient                 remoteexecution.ExecutionClient
	requestTranslator               translator.RequestTranslator
}

// NewFetchingFetcher creates a new Fetcher that is capable of itself fetching
// resources from other places (as defined in the qualifier_translator).
func NewFetchingFetcher(fetcher Fetcher, pusher remoteasset.PushServer, client grpc.ClientConnInterface) Fetcher {
	return &fetchingFetcher{
		fetcher:             fetcher,
		pusher:              pusher,
		actionCacheClient:   remoteexecution.NewActionCacheClient(client),
		qualifierTranslator: qualifier.SimpleQualifierTranslator{},
	}
}

func (ff *fetchingFetcher) FetchBlob(ctx context.Context, req *remoteasset.FetchBlobRequest) (*remoteasset.FetchBlobResponse, error) {
	resp, err := ff.FetchBlob(ctx, req)
	if err == nil {
		return resp
	}

	command := qualifier.SimpleQualifierTranslator{}.QualifierToCommand(req.Qualifiers)
	commandDigest, err := translator.ProtoToDigest(&command)
	if err != nil {
		return nil, err
	}

	action := remoteexecution.Action{
		CommandDigest:   commandDigest,
		InputRootDigest: EmptyDigest,
	}
	actionDigest, err := translator.ProtoToDigest(&action)
	if err != nil {
		return nil, err
	}

	actionPb, err := proto.Marshal(&action)
	if err != nil {
		return nil, err
	}

	commandPb, err := proto.Marshal(&command)
	if err != nil {
		return nil, err
	}

	_, err = ff.contentAddressableStorageClient.BatchUpdateBlobs(ctx, &remoteexecution.BatchUpdateBlobsRequest{
		InstanceName: req.InstanceName,
		Requests: []*remoteexecution.BatchUpdateBlobsRequest_Request{
			&remoteexecution.BatchUpdateBlobsRequest_Request{
				Digest: actionDigest,
				Data:   actionPb,
			},
			&remoteexecution.BatchUpdateBlobsRequest_Request{
				Digest: commandDigest,
				Data:   commandPb,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = ff.actionCacheClient.UpdateActionResult(ctx, &remoteexecution.UpdateActionResultRequest{
		InstanceName: req.InstanceName,
		ActionDigest: actionDigest,
		ActionResult: &actionResult,
	})
	if err != nil {
		return nil, err
	}

	stream := ff.executionClient.Execute(ctx, remoteexecution.ExecuteRequest{
		ActionDigest: actionDigest,
	})

	for {
		operation, err := stream.Recv()
		if err != nil {
			return nil, err
		}

		if operation.done {
			switch operation.isOperation_Result.(type) {
			case operation.Operation_Error:
				return nil, nil
			case operation.Operation_Result:
			}
		}

		longrunning.
	}

	return ff.pusher.PushBlob(ctx, req)
}

func (ff *fetchingFetcher) FetchDirectory(ctx context.Context, req *remoteasset.FetchDirectoryRequest) (*remoteasset.FetchDirectoryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "FetchDirectory not implemented yet!")
}

func (ff *fetchingFetcher) CheckQualifiers(qualifiers qualifier.Set) qualifier.Set {
	return qualifier.Set{}
}
