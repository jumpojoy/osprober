Osprober is a minitoring software that provides extensions for Cloudprober(https://github.com/cloudprober/cloudprober)
to provide integration with OpenStack clouds(https://docs.openstack.org/).

## Motivation

One of major SLA for public/private clouds is network accessibility of its workloads.
The biggest channge in setupping monitoring for workloads in the cloud is the requirement
to open access from monitoring software to workload via monitoring protocols that are used
for example ICMP. This is not always possible. For those cases low level monitoring via
ARP may be used. This project implements missing probes in Cloudprobber that allow to
monitor workloads absed on ARP.

## Features

- Probes extensions:

  - Arping extension. Allows to monitor hosts availability via ARP protocol. Requires that
    monitoring software is directly connected to hosts network. 

- Surfacer extensions:

  - Formated File extension. Store monitoring metrics in file with speceific format. Only
    json format is supported at the moment.

## Examples

sudo ./osprober --config_file examples/file_based_targets/cloudprober.cfg   --formated_file_metrics=/tmp/foo.txt
