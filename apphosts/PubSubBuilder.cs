using System.Net;
using Microsoft.Extensions.Configuration;
using Aspire.Hosting;
using Aspire.Hosting.ApplicationModel;

namespace apphosts;

public sealed record PubSubTopicResource(
    string ProjectID
    , string TopicID
    , IResourceBuilder<ExecutableResource> Creator
    , EndpointReference PubsubEndpoint
);

public sealed class PubSubSubscriptionBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<ContainerResource> pubsub
    , PubSubTopicResource pubsubTopic
)
{
    public PubSubTopicResource Topic { get; } = pubsubTopic;

    public IResourceBuilder<ExecutableResource> PushSubscription(string subscriptionID, EndpointReference pushEndpoint, string pushVerificationToken)
    {
        pubsub.WithReference(pushEndpoint);

        return builder.AddGoApp(
            name: subscriptionID,
            appDirectory: "../tools/pubsub",
            packagePath: "./cmd/subscription"
        )
        .WithReference(Topic.PubsubEndpoint)
        .WithEnvironment(context =>
        {
            context.EnvironmentVariables["PUBSUB_EMULATOR_HOST"] = Topic.PubsubEndpoint.Property(EndpointProperty.HostAndPort);
            context.EnvironmentVariables["PUBSUB_PROJECT_ID"] = Topic.ProjectID;
            context.EnvironmentVariables["PUBSUB_TOPIC_ID"] = Topic.TopicID;
            context.EnvironmentVariables["PUBSUB_SUBSCRIPTION_ID"] = subscriptionID;
            context.EnvironmentVariables["PUBSUB_PUSH_ENDPOINT"] = ReferenceExpression.Create($"{pushEndpoint.Property(EndpointProperty.Url)}?token={pushVerificationToken}");
        })
        .WaitFor(pubsub)
        .WaitForCompletion(Topic.Creator)
        ;
    }
}

public sealed class PubSubTopicBuilder(
    IDistributedApplicationBuilder builder
    , IResourceBuilder<ContainerResource> pubsub
    , string projectID
    , EndpointReference pubsubEndpoint
)
{
    public PubSubSubscriptionBuilder Topic(string topicID)
    {
        var creator = builder.AddGoApp(
            name: topicID,
            appDirectory: "../tools/pubsub",
            packagePath: "./cmd/topic"
        )
        .WithReference(pubsubEndpoint)
        .WithEnvironment(context =>
        {
            context.EnvironmentVariables["PUBSUB_EMULATOR_HOST"] = pubsubEndpoint.Property(EndpointProperty.HostAndPort);
            context.EnvironmentVariables["PUBSUB_PROJECT_ID"] = projectID;
            context.EnvironmentVariables["PUBSUB_TOPIC_ID"] = topicID;
        })
        .WaitFor(pubsub)
        ;
        return new(
            builder: builder,
            pubsub: pubsub,
            pubsubTopic: new(
                ProjectID: projectID,
                TopicID: topicID,
                Creator: creator,
                PubsubEndpoint: pubsubEndpoint
            )
        );
    }
}

public sealed class PubSubBuilder(
    IDistributedApplicationBuilder builder
)
{
    public PubSubTopicBuilder PubSub(int port = 8085, string projectID = "jobdeck-local")
    {
        var pubsub = builder.AddContainer(
                name: "pubsub",
                image: "gcr.io/google.com/cloudsdktool/google-cloud-cli",
                tag: "emulators"
            )
            .WithEntrypoint("gcloud")
            .WithArgs(
                "beta", "emulators", "pubsub", "start",
                $"--project={projectID}",
                $"--host-port=0.0.0.0:{port}"
            )
            .WithHttpEndpoint(name: "http", targetPort: port)
            ;
        var pubsubEndpoint = pubsub.GetEndpoint("http");
        return new(
            builder: builder,
            pubsub: pubsub,
            projectID: projectID,
            pubsubEndpoint: pubsubEndpoint
        );
    }
}
