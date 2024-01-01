package main

import (
	"context"
	"flag"
	"github.com/cloudprober/cloudprober"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/jumpojoy/osprober/arping"
	"github.com/jumpojoy/osprober/surfacers/formated_file"
	surfacerpb "github.com/jumpojoy/osprober/surfacers/formated_file/proto"
)

func main() {
	// TODO(vsaienko): move to cloudprober surfacer config when PR is resolved
	// https://github.com/cloudprober/cloudprober/issues/661
	ff_metrics := flag.String("formated_file_metrics", "", "Where to store file map metrics.")
	flag.Parse()

	var log = logger.New()

	// Register stubby probe type
	probes.RegisterProbeType(int(arping.E_ArpingProbe.TypeDescriptor().Number()),
		func() probes.Probe { return &arping.Probe{} })

	s, _ := formated_file.New(&surfacerpb.SurfacerConf{FilePath: ff_metrics}, log)
        surfacers.Register("formated_file", s)

	if err := cloudprober.Init(); err != nil {
		log.Criticalf("Error initializing osprober. Err: %v", err)
	}

	cloudprober.Start(context.Background())

	// Wait forever
	select {}
}
