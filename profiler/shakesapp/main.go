// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Shakesapp is a web application which starts up a server that can be
// queried to determine how many times a string appears in the works of
// Shakespeare and then sends requests to that server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/GoogleCloudPlatform/golang-samples/profiler/shakesapp/shakesapp"
	"github.com/pyroscope-io/client/pyroscope"
)

var (
	projectID        = flag.String("project_id", "", "project ID to run profiler with; only required when running outside of GCP.")
	version          = flag.String("version", "original", "version to run profiler with")
	port             = flag.Int("port", 7788, "service port")
	numReqs          = flag.Int("num_requests", 20, "number of requests to simulate")
	concurrency      = flag.Int("concurrency", 1, "number of requests to run in parallel")
	numRounds        = flag.Int("num_rounds", 0, "number of simulation rounds (0 for infinite)")
	enableHeap       = flag.Bool("heap", false, "enable heap profile collection")
	enableHeapAlloc  = flag.Bool("heap_alloc", false, "enable heap allocation profile collection")
	enableThread     = flag.Bool("thread", false, "enable thread profile collection")
	enableContention = flag.Bool("contention", false, "enable contention profile collection")
)

func main() {
	flag.Parse()

	// if err := profiler.Start(profiler.Config{
	// 	Service:              "shakesapp",
	// 	ServiceVersion:       *version,
	// 	ProjectID:            *projectID,
	// 	NoHeapProfiling:      !*enableHeap,
	// 	NoAllocProfiling:     !*enableHeapAlloc,
	// 	NoGoroutineProfiling: !*enableThread,
	// 	MutexProfiling:       *enableContention,
	// 	DebugLogging:         true,
	// }); err != nil {
	// 	log.Fatalf("Failed to start profiler: %v", err)
	// }

	pyroscope.Start(pyroscope.Config{
		ApplicationName: "shakesapp",

		// Static IP of pyroscope VM
		ServerAddress: "http://10.128.0.2:4040",

		// you can disable logging by setting this to nil
		Logger: pyroscope.StandardLogger,

		// optionally, if authentication is enabled, specify the API key:
		// AuthToken:    os.Getenv("PYROSCOPE_AUTH_TOKEN"),

		// you can provide static tags via a map:
		Tags: map[string]string{
			"hostname": os.Getenv("HOSTNAME"),
			"version":  *version,
		},

		ProfileTypes: []pyroscope.ProfileType{
			// these profile types are enabled by default:
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			// these profile types are optional:
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})

	server := grpc.NewServer()
	shakesapp.RegisterShakespeareServiceServer(server, shakesapp.NewServer())
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	go server.Serve(lis)

	ctx := context.Background()
	for i := 1; *numRounds == 0 || i <= *numRounds; i++ {
		start := time.Now()
		log.Printf("Simulating client requests, round %d", i)
		if err := shakesapp.SimulateClient(ctx, fmt.Sprintf(":%d", *port), *numReqs, *concurrency); err != nil {
			log.Fatalf("Failed to simulate client requests: %v", err)
		}
		delta := time.Since(start).Round(10 * time.Millisecond)
		log.Printf("Simulated %d requests in %s, rate of %f reqs / sec", *numReqs, delta, float64(*numReqs)/delta.Seconds())
	}
}
