[![CircleCI](https://circleci.com/gh/giantswarm/aws-rolling-node-operator.svg?style=shield)](https://circleci.com/gh/giantswarm/aws-rolling-node-operator)

# Operator for rolling nodes

The `aws-rolling-node-operator` reconciles on `AWSCluster`, `AWSControlplane` and `AWSMachineDeployment` Custom Resources to trigger an EC2 instance refresh on the given Auto Scaling groups. After a Auto scaling group got refreshed the operator won't allow to refresh instances again within `30` minutes.

Setting the annotation `alpha.giantswarm.io/instance-refresh: true` will refresh (terminate and start new EC2 instances) EC2 instances based on the Custom Reource:

- `AWSCluster` CR - Refreshes all EC2 instances for the Control Plane and node pools.
- `AWSControlplane` CR - Refreshes all EC2 instances for the Control Plane.
- `AWSMachineDeployment` CR - Refreshes all EC2 instances for a specific node pool.

Once the EC2 instance refresh is finished, the `aws-rolling-node-operator` will remove the annotation from Custom Resource and will send a Kubernetes Event on the Custom Resource, e.g.:

```yaml
Events:
  Type    Reason              Age                     From                                           Message
  ----    ------              ----                    ----                                           -------
  Normal  InstancesRefreshed  10m                     aws-machinedeployment-node-rolling-controller  Refreshed all worker instances.
```

Additionally annotations which can be set:

`alpha.aws.giantswarm.io/instance-refresh-min-healthy-percentage` - Sets the amount of capacity which must remain healthy inside the Auto Scaling group. The value is expressed as a percentage of the desired capacity of the Auto Scaling group (rounded up to the nearest integer). The default is 90. Setting the minimum healthy percentage to 100 percent limits the rate of replacement to one instance at a time. In contrast, setting it to 0 percent has the effect of replacing all instances at the same time.

`alpha.aws.giantswarm.io/instance-warmup-seconds` - The instance warmup is the time period from when a new instance's state changes to InService to when it can receive traffic. During an instance refresh, Amazon EC2 Auto Scaling does not immediately move on to the next replacement after determining that a newly launched instance is healthy. It waits for the warm-up period that you specified before it moves on to replacing other instances. This can be helpful when your application takes time to initialize itself before it starts to serve traffic. The default is 0.

`alpha.aws.giantswarm.io/cancel-instance-refresh` - This will immediately cancel the current instance refresh. It stops replacing nodes which haven’t been rolled so far.
