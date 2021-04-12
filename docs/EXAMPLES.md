### Examples

### [Using HTTP for input and output](examples/101-http-pipeline.yaml)

This is an example of send and receive messages using HTTP.

To receive a message, you must expose a HTTP endpoint on http://localhost:8080/messages. Each message will
be passed as the body of a single HTTP POST request.

To send a message, send a HTTP post to http://localhost:3569/messages.

The image `argoproj/dataflow-runner` has a convenience `cat` sub-command that just copies the input to the output.


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/101-http-pipeline.yaml
```

### [Two nodes pipeline](examples/101-two-node-pipeline.yaml)

This example shows a example of having two nodes in a pipeline.

While they read from Kafka, they are connected by a NATS Streaming subject.


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/101-two-node-pipeline.yaml
```

### [Filter messages](examples/102-filter-pipeline.yaml)

This is an example of built-in filtering.

Filters are written using expression syntax and must return a boolean.

They have a single variable, `msg`, which is a byte array.

[Learn about expressions](../docs/EXPRESSIONS.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/102-filter-pipeline.yaml
```

### [Map messages](examples/102-map-pipeline.yaml)

This is an example of built-in mapping.

Maps are written using expression syntax and must return a byte array.

They have a single variable, `msg`, which is a byte array.

[Learn about expressions](../docs/EXPRESSIONS.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/102-map-pipeline.yaml
```

### [Using replicas and auto-scaling](examples/103-replicas-pipeline.yaml)

This is an example of having multiple replicas for a single step.

Replicas are automatically scaled up and down depending on the number of messages pending processing.

The ratio is defined as the number of pending messages per replica:

```
replicas = pending / ratio
```

The number of replicas will not scale beyond the min/max bounds (except when *peeking*, see below):

```
min <= replicas <= max
```

* `min` is used as the initial number of replicas.
* If `ratio` is undefined no scaling can occur; `max` is meaningless.
* If `ratio` is defined but `max` is not, the step may scale to infinity.
* If `max` and `ratio` are undefined, then the number of replicas is `min`.
* In this example, because the ratio is 1000, if 2000 messages pending, two replicas will be started.
* To prevent scaling up and down repeatedly - scale up or down occurs a maximum of once a minute.
* The same message will not be send to two different replicas.

### Scale-To-Zero and Peeking

You can scale to zero by setting `min: 0`. The number of replicas will start at zero, and periodically be scaled
to 1  so it can "peek" the the message queue. The number of pending messages is measured and the target number
of replicas re-calculated.


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/103-replicas-pipeline.yaml
```

### [Go 1.16 handler](examples/104-go1-16-pipeline.yaml)

This example of Go 1.16 handler.

[Learn about handlers](../docs/HANDLERS.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/104-go1-16-pipeline.yaml
```

### [Java 16 handler](examples/104-java16-pipeline.yaml)

This example is of the Java 16 handler.

[Learn about handlers](../docs/HANDLERS.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/104-java16-pipeline.yaml
```

### [Python 3.9 handler](examples/104-python-pipeline.yaml)

This example is of the Python 3.9 handler.

[Learn about handlers](../docs/HANDLERS.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/104-python-pipeline.yaml
```

### [Git handler](examples/106-git-pipeline.yaml)

This example of a pipeline using Git.

The Git handler allows you to check your application source code into Git. Dataflow will checkout and build
your code when the step starts.

[Learn about Git steps](../docs/GIT.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/106-git-pipeline.yaml
```

### [Runs to completion](examples/107-completion-pipeline.yaml)

This example shows a pipelne running to completion.

A pipeline that run to completion (aka "terimanting") is one that will finish.

For a pipeline to terminate one of two things must happen:

* Every steps exits successfully (i.e. with exit code 0).
* One step exits successfully, and is marked with `terminator: true`. When this happens, all other steps are killed.


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/107-completion-pipeline.yaml
```

### [Using FIFOs for input and outputs](examples/108-fifos-pipeline.yaml)

This example use named pipe to send and receive messages.

Two named pipes are made available:

* The container can read lines from `/var/run/argo-dataflow/in`. Each line will be a single message.
* The contain can write to `/var/run/argo-dataflow/out`. Each line MUST be a single message.

You MUST escape new lines.


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/108-fifos-pipeline.yaml
```

### [Group messages](examples/109-group-pipeline.yaml)

This is an example of built-in grouping.

There are four mandatory fields:

* `key` A string expression that returns the message's key
* `endOfGroup` A boolean expression that returns whether or not to end group and send the group messages onwards.
* `format` What format the grouped messages should be in.
* `storage` Where to store messages ready to be forwarded.

[Learn about expressions](../docs/EXPRESSIONS.md)

### Storage

Storage can either be:

* An ephemeral volume - you don't mind loosing some or all messages (e.g. development or pre-production).
* A persistent volume - you want to be to recover (e.g. production).


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/109-group-pipeline.yaml
```

### [Vetinary](examples/201-vetinary-pipeline.yaml)

This pipeline processes pets (cats and dogs).


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/201-vetinary-pipeline.yaml
```

### [Word count](examples/201-word-count-pipeline.yaml)

This pipeline count the number of words in a document, not the number of count of each word as you might expect.

It also shows an example of a pipelines terminates based on a single step's status.


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/201-word-count-pipeline.yaml
```

### [Erroring](examples/301-erroring-pipeline.yaml)

This example creates errors randomly


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/301-erroring-pipeline.yaml
```

### [Default Kafka config](examples/dataflow-kafka-default-secret.yaml)

This is an example of providing a namespace named Kafka configuration.

The secret must be named `dataflow-kafka-${name}`.

[Learn about configuration](../docs/CONFIGURATION.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/dataflow-kafka-default-secret.yaml
```

### [Default NATS Streaming (STAN) configuration](examples/dataflow-stan-default-secret.yaml)

This is an example of providing a namespace named NATS Streaming configuration.

The secret must be named `dataflow-nats-${name}`.

[Learn about configuration](../docs/CONFIGURATION.md)


```
kubectl apply -f https://raw.githubusercontent.com/argoproj-labs/argo-dataflow/main/examples/dataflow-stan-default-secret.yaml
```
