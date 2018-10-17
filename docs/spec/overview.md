# Resource Types

The API defines and provides a complete implementation for
[Subscription](spec.md#kind-subscription), and abstract resource definitions
for [Sources](spec.md#kind-source), [Channels](spec.md#kind-channel), and
[ClusterProvisioner](spec.md#kind-clusterprovisioner) which may be fulfilled by multiple
backing implementations (much like the Kubernetes Ingress resource).

- A **Subscription** describes the transformation of an event and optional
  forwarding of a returned event.

- A **Source** allows an incoming events from an external system to be
  _Subscribable_. A _Subscription_ is used to connect these events to
  subsequent processing steps.

- A **Channel** provides event persistance and fanout of events from a
  well-known input address to multiple outputs described by _Subscriptions_.

<!-- This image is sourced from https://drive.google.com/open?id=10mmXzDb8S_4_ZG_hcBr7s4HPISyBqcqeJLTXLwkilRc -->

![Resource Types Overview](images/resource-types-overview.svg)

- **Provisioners** implement strategies for realizing backing resources for
  different implementations of _Sources_ and _Channels_ currently active in the
  eventing system.

<!-- This image is sourced from https://drive.google.com/open?id=1o_0Xh5VjwpQ7Px08h_Q4qnaOdMjt4yCEPixRFwJQjh8 -->

![Resource Types Provisioners](images/resource-types-provisioner.svg)

With extendibility and compostability as a goal of Knative Eventing, the
eventing API defines several resources that can be reduced down to a well
understood contracts. These eventing resource interfaces may be fulfilled by
other Kubernetes objects and then composed in the same way as the concreate
objects. The interfaces are ([Sinkable](interfaces.md#sinkable),
[Subscribable](interfaces.md#subscribable),
[Channelable](interfaces.md#channelable),
[Targetable](interfaces.md#targetable)). For more details, see
[Interface Contracts](interfaces.md).

## Subscription

**Subscriptions** describe a flow of events from one event producer or
forwarder (typically, _Source_ or _Channel_) to the next (typically, a
_Channel_) through transformations (such as a Knative Service which processes
CloudEvents over HTTP). A _Subscription_ controller resolves the addresses of
transformations (`call`) and destination storage (`result`) through the
_Targetable_ and _Sinkable_ interface contracts, and writes the resolved
addresses to the _Subscribable_ `from` resource. _Subscriptions_ do not need to
specify both a transformation and a storage destination, but at least one must
be provided.

All event delivery linkage from a **Subscription** is 1:1 – only a single
`from`, `call`, and `result` may be provided.

For more details, see [Kind: Subscription](spec.md#kind-subscription).

## Source

**Source** represents incoming events from an external system, such as object
creation events in a specific storage bucket, database updates in a particular
table, or Kubernetes resource state changes. Because a _Source_ represents an
external system, it only produces events (and is therefore _Subscribable_ by
_Subscriptions_). _Source_ may include parameters such as specific resource
names, event types, or credentials which should be used to establish the
connection to the external system. The set of allowed configuration parameters
is described by the _Provisioner_ which is referenced by the _Source_.

Event selection on a _Source_ is 1:N – a single _Source_ may fan out to
multiple _Subscriptions_.

For more details, see [Kind: Source](spec.md#kind-source).

## Channel

**Channel** provides an at least once event delivery mechanism which can fan
out received events to multiple destinations via _Subscriptions_. A _Channel_
has a single inbound _Sinkable_ interface which may accept events from multiple
_Subscriptions_ or even direct delivery from external systems. Different
_Channels_ may implement different degrees of persistence. Event delivery order
is dependent on the backing implementation of the _Channel_ provided by the
_Provisioner_.

Event selection on a _Channel_ is 1:N – a single _Channel_ may fan out to
multiple _Subscriptions_.

See [Kind: Channel](spec.md#kind-channel).

## Provisioner

**ClusterProvisioners** are responsible for realizing _Sources_ and _Channels_ when required.
Collectively they define the set of _Sources_ and _Channels_ which can be deployed into Knative.
_ClusterProvisioners_ are not required to instantiate all dependent resources of a _Source_ or a _Channel_ but may instead interact with pre-existing systems.
For example, the provisioner of a _Channel_ may interact with a message broker to create the required messaging endpoints, such as GC PubSub topics, rather than instantiating the message broker itself.

_Sources_ and _Channels_ reference a _ClusterProvisioner_ and can supply input arguments, eg. the URL of a message broker that a _Source_ will consume messages from.
In future, the _ClusterProvisioner_ may provide defaults for arguments and validate arguments against a JSON Schema that it holds.

As an example, consider a _Source_ which connects Knative Eventing to an external email system.
In order to achieve this a developer would write the application code for connecting via a suitable protocol (IMAP, SMTP, POP3,...) and package this into an image.
They would also create a _ClusterProvisioner_ which is capable of deploying the image into Kubernetes (for simple _Sources_ generic _ClusterProvisioners_ may be available).
The _ClusterProvisioner_ can now be used to provision the _Source_ by other users.
When a user deploys the _Source_ they can provide connection arguments which the _ClusterProvisioner_ will use to configure the image (eg. mail servers, credentials, etc.).
When deploying the _Source_ an optional reference to an existing _Channel_ can be specified but if this is empty a new one will be created using the cluster/namespace defaults.

_ClusterProvisioners_ do not directly handle events.
One _ClusterProvisioner_ must be able to realize multiple _Channels_ or _Sources_ - they are 1:N.

For more details, see [Kind: ClusterProvisioner](#kind-clusterprovisioner)

---

_Navigation_:

- [Motivation and goals](motivation.md)
- **Resource type overview**
- [Interface contracts](interfaces.md)
- [Object model specification](spec.md)
