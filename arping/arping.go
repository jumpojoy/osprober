package arping

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

var (
    timeout    = time.Duration(1 * time.Second)
)

// SetTimeout sets ping timeout
func SetTimeout(t time.Duration) {
	timeout = t
}

// Get destination MAC address for endpoint 
func getEndpointMac(target endpoint.Endpoint) (net.HardwareAddr, error) {
	val, ok := target.Labels["mac"]
	if ok {
		mac, err := net.ParseMAC(val)
		if err != nil {
		    return nil, err
		}
	    return mac, nil
	}
	return nil, nil
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	c       *ProbeConf
	//targets []endpoint.Endpoint
	targets []endpoint.Endpoint
	opts    *options.Options

	res map[string]*metrics.EventMetrics // Results by target
	l   *logger.Logger
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*ProbeConf)
	if !ok {
		return fmt.Errorf("Not a my probe config")
	}
	p.c = c
	p.name = name
	p.opts = opts
	p.l = opts.Logger

	p.res = make(map[string]*metrics.EventMetrics)
	return nil
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	probeTicker := time.NewTicker(p.opts.Interval)

	for {
		select {
		case <-ctx.Done():
			probeTicker.Stop()
			return
		case <-probeTicker.C:
			var oldTargets []endpoint.Endpoint = p.targets
			p.targets = p.opts.Targets.ListEndpoints()

			p.initProbeMetrics()
			p.clenupTargets(oldTargets)
			probeCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(p.opts.Timeout))
			p.runProbe(probeCtx, dataChan)
			p.exportMetrics(dataChan)
			cancelFunc()
		}
	}
}

// initProbeMetrics initializes missing probe metrics.
func (p *Probe) initProbeMetrics() {
	for _, target := range p.targets {
		if p.res[target.Name] != nil {
			continue
		}
		var latVal metrics.Value
		if p.opts.LatencyDist != nil {
			latVal = p.opts.LatencyDist.Clone()
		} else {
			latVal = metrics.NewFloat(0)
		}
		p.res[target.Name] = metrics.NewEventMetrics(time.Now()).
			AddMetric("total", metrics.NewInt(0)).
			AddMetric("success", metrics.NewInt(0)).
			AddMetric("latency", latVal).
			AddLabel("ptype", "arping").
			AddLabel("probe", p.name).
			AddLabel("dst", target.Name)
	}
}

// send event for deleted targets, tootal == -1 means it was deleted
func (p *Probe) clenupTargets(targets []endpoint.Endpoint) {
        for _, t := range targets {
		var isExists bool = false
                for _, n := range p.targets {
			if n.Name == t.Name {
				isExists = true
                                break
                        }
                }
                if !isExists {
			delete(p.res, t.Name)
                }

        }

}

// runProbeForTarget runs probe for a single target.
func (p *Probe) runProbeForTarget(ctx context.Context, target endpoint.Endpoint) error {
	p.l.Debugf("Checking target %s", target)
	SetTimeout(1*time.Second)

        dstIP := target.IP
        dstMac, err := getEndpointMac(target)
	if err != nil {
		p.l.Errorf("Failed to find destination MAC for endpoint  %s", target.Name)
	 	return nil
	}
        if dstMac == nil{
	       dstMac = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}

        iface, err := findUsableInterfaceForNetwork(dstIP)
        if err != nil {
		p.l.Errorf("Failed to find interface to monitor target %s, %s", target.Name, target.IP.String())
                return nil
        }


	srcMac := iface.HardwareAddr
	srcIP, err := findIPInNetworkFromIface(dstIP, *iface)
	if err != nil {
		p.l.Errorf("Failed to find address for interface.%s: %s", iface.Name, err.Error())
		return err
	}
	//p.l.Errorf("srcMac %s srcIP %s dstMac %s", srcMac, srcIP, dstMac)
	//srcMac, err := net.ParseMAC("fa:16:3e:78:b1:9d")
	//srcIP := net.ParseIP("192.168.50.1")
	request := newArpRequest(srcMac, srcIP, dstMac, dstIP)

	sock, err := initialize(*iface)
	if err != nil {
		p.l.Errorf("Failed to initialize socket on interface %s: %s", iface.Name, err.Error())
		return err
	}
	defer sock.deinitialize()

	if _, err := sock.send(request); err != nil {
		p.l.Errorf("Failed to send arp request: %s", err.Error())
		return err
	} else {
	    for {
		// receive arp response
		response, _, err := sock.receive()
		if err != nil {
			p.l.Debugf("Did not get reply for %s %s %s", target.Name, target.IP.String(), err.Error())
			return err
		}

		if response.IsResponseOf(request) {
			return nil
		}
	    }
	}

	select {
	case <-time.After(timeout):
		sock.deinitialize()
		p.l.Errorf("Timed out waiting arp reply for target %s: %s", target.Name, target.IP.String())
		return fmt.Errorf("Timed out waiting arp reply")
	}
	return nil
}

// Export metrics to surfacer
func (p *Probe) exportMetrics(dataChan chan *metrics.EventMetrics){
        for _, target := range p.targets {
		// Update pod labels, do transformation from
                // target labels.
                for _, al := range p.opts.AdditionalLabels {
                        al.UpdateForTarget(target, target.IP.String(), 0)
                }
	        p.opts.RecordMetrics(target, p.res[target.Name], dataChan)
        }
}

// runProbe runs probe for all targets and update EventMetrics.
func (p *Probe) runProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.targets = p.opts.Targets.ListEndpoints()

	var wg sync.WaitGroup
	maxWorkers := p.c.GetMaxWorkers()
	p.l.Debugf("Running cycle for probe targets %s with maxWorkers %d", p.name, maxWorkers)

	semaphore := make(chan struct{}, maxWorkers)
	start := time.Now()

	for _, target := range p.targets {

		p.l.Debugf("Running probe for target %s, %s", target.Name, target.IP)
		wg.Add(1)
		semaphore <- struct{}{} // acquire semaphore

		go func(target endpoint.Endpoint, em *metrics.EventMetrics) {
			defer wg.Done()
			start := time.Now()
			em.Timestamp = start
			em.Metric("total").(*metrics.Int).Inc()
			err := p.runProbeForTarget(ctx, target) // run probe just for a single target
			<-semaphore // release semaphore
			if err != nil {
				p.l.Debugf(err.Error())
				return
			}
			em.Metric("success").(*metrics.Int).Inc()
			em.Metric("latency").(metrics.LatencyValue).AddFloat64(time.Since(start).Seconds() / p.opts.LatencyUnit.Seconds())
		}(target, p.res[target.Name])
	}
	wg.Wait()
	p.l.Debugf("Finished cycle for probe targets %s", p.name)
	elapsed_seconds := int(time.Since(start).Seconds())
        p.l.Debugf("Took %d to check all targets.", elapsed_seconds)
}
