var builder = DistributedApplication.CreateBuilder(args);

// Each startup test adds only the resources needed for its scenario.
builder.Build().Run();
