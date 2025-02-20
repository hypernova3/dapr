/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/util/sets"

	commonv1pb "github.com/dapr/dapr/pkg/proto/common/v1"
	pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
)

const (
	appPort     = 3000
	pubsubName  = "messagebus"
	pubsubTopic = "pubsub-routing-grpc"

	pathA = "myevent.A"
	pathB = "myevent.B"
	pathC = "myevent.C"
	pathD = "myevent.D"
	pathE = "myevent.E"
	pathF = "myevent.F"
)

type routedMessagesResponse struct {
	RouteA []string `json:"route-a"`
	RouteB []string `json:"route-b"`
	RouteC []string `json:"route-c"`
	RouteD []string `json:"route-d"`
	RouteE []string `json:"route-e"`
	RouteF []string `json:"route-f"`
}

var (
	// using sets to make the test idempotent on multiple delivery of same message.
	routedMessagesA sets.String
	routedMessagesB sets.String
	routedMessagesC sets.String
	routedMessagesD sets.String
	routedMessagesE sets.String
	routedMessagesF sets.String
	lock            sync.Mutex
)

// server is our user app.
type server struct{}

func main() {
	log.Printf("Initializing grpc")

	/* #nosec */
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", appPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	lock.Lock()
	initializeSets()
	lock.Unlock()

	/* #nosec */
	s := grpc.NewServer()
	pb.RegisterAppCallbackServer(s, &server{})

	log.Println("Client starting...")

	// Stop the gRPC server when we get a termination signal
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT) //nolint:staticcheck
	go func() {
		// Wait for cancelation signal
		<-stopCh
		log.Println("Shutdown signal received")
		s.GracefulStop()
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	log.Println("App shut down")
}

// initialize all the sets for a clean test.
func initializeSets() {
	// initialize all the sets.
	routedMessagesA = sets.NewString()
	routedMessagesB = sets.NewString()
	routedMessagesC = sets.NewString()
	routedMessagesD = sets.NewString()
	routedMessagesE = sets.NewString()
	routedMessagesF = sets.NewString()
}

// This method gets invoked when a remote service has called the app through Dapr.
// The payload carries a Method to identify the method, a set of metadata properties and an optional payload.
func (s *server) OnInvoke(ctx context.Context, in *commonv1pb.InvokeRequest) (*commonv1pb.InvokeResponse, error) {
	reqID := "s-" + uuid.New().String()
	if in.HttpExtension != nil && in.HttpExtension.Querystring != "" {
		qs, err := url.ParseQuery(in.HttpExtension.Querystring)
		if err == nil && qs.Has("reqid") {
			reqID = qs.Get("reqid")
		}
	}

	log.Printf("(%s) Got invoked method %s", reqID, in.Method)

	lock.Lock()
	defer lock.Unlock()

	respBody := &anypb.Any{}
	switch in.Method {
	case "getMessages":
		respBody.Value = s.getMessages(reqID)
	case "initialize":
		initializeSets()
	}

	return &commonv1pb.InvokeResponse{Data: respBody, ContentType: "application/json"}, nil
}

func (s *server) getMessages(reqID string) []byte {
	resp := routedMessagesResponse{
		RouteA: routedMessagesA.List(),
		RouteB: routedMessagesB.List(),
		RouteC: routedMessagesC.List(),
		RouteD: routedMessagesD.List(),
		RouteE: routedMessagesE.List(),
		RouteF: routedMessagesF.List(),
	}

	rawResp, _ := json.Marshal(resp)
	log.Printf("(%s) getMessages response: %s", reqID, string(rawResp))
	return rawResp
}

// Dapr will call this method to get the list of topics the app wants to subscribe to. In this example, we are telling Dapr.
// To subscribe to a topic named TopicA.
func (s *server) ListTopicSubscriptions(ctx context.Context, in *emptypb.Empty) (*pb.ListTopicSubscriptionsResponse, error) {
	log.Println("List Topic Subscription called")
	return &pb.ListTopicSubscriptionsResponse{
		Subscriptions: []*commonv1pb.TopicSubscription{
			{
				PubsubName: pubsubName,
				Topic:      pubsubTopic,
				Routes: &commonv1pb.TopicRoutes{
					Rules: []*commonv1pb.TopicRule{
						{
							Match: `event.type == "myevent.C"`,
							Path:  pathC,
						},
						{
							Match: `event.type == "myevent.B"`,
							Path:  pathB,
						},
					},
					Default: pathA,
				},
			},
		},
	}, nil
}

// This method is fired whenever a message has been published to a topic that has been subscribed. Dapr sends published messages in a CloudEvents 1.0 envelope.
func (s *server) OnTopicEvent(ctx context.Context, in *pb.TopicEventRequest) (*pb.TopicEventResponse, error) {
	lock.Lock()
	defer lock.Unlock()

	reqID := uuid.New().String()
	log.Printf("(%s) Message arrived - Topic: %s, Message: %s, Path: %s", reqID, in.Topic, string(in.Data), in.Path)

	var set *sets.String
	switch in.Path {
	case pathA:
		set = &routedMessagesA
	case pathB:
		set = &routedMessagesB
	case pathC:
		set = &routedMessagesC
	case pathD:
		set = &routedMessagesD
	case pathE:
		set = &routedMessagesE
	case pathF:
		set = &routedMessagesF
	default:
		log.Printf("(%s) Responding with DROP. in.Path not found", reqID)
		// Return success with DROP status to drop message.
		return &pb.TopicEventResponse{
			Status: pb.TopicEventResponse_DROP, //nolint:nosnakecase
		}, nil
	}

	msg := string(in.Data)

	set.Insert(msg)

	log.Printf("(%s) Responding with SUCCESS", reqID)
	return &pb.TopicEventResponse{
		Status: pb.TopicEventResponse_SUCCESS, //nolint:nosnakecase
	}, nil
}

// Dapr will call this method to get the list of bindings the app will get invoked by. In this example, we are telling Dapr.
// To invoke our app with a binding named storage.
func (s *server) ListInputBindings(ctx context.Context, in *emptypb.Empty) (*pb.ListInputBindingsResponse, error) {
	log.Println("List Input Bindings called")
	return &pb.ListInputBindingsResponse{}, nil
}

// This method gets invoked every time a new event is fired from a registered binding. The message carries the binding name, a payload and optional metadata.
func (s *server) OnBindingEvent(ctx context.Context, in *pb.BindingEventRequest) (*pb.BindingEventResponse, error) {
	log.Printf("Invoked from binding: %s", in.Name)
	return &pb.BindingEventResponse{}, nil
}
