# syntax=docker/dockerfile:1

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src

# Restore first so NuGet packages remain cached when only application code changes.
COPY NuGet.Config ./
COPY gen/csharp/Jobdeck.Proto/Jobdeck.Proto.csproj gen/csharp/Jobdeck.Proto/
COPY services/ui/ui.csproj services/ui/
RUN dotnet restore services/ui/ui.csproj

COPY gen/csharp/ gen/csharp/
COPY services/ui/ services/ui/
RUN dotnet publish services/ui/ui.csproj --no-restore --configuration Release --output /app/publish

FROM mcr.microsoft.com/dotnet/aspnet:10.0
WORKDIR /app
COPY --from=build /app/publish .
ENV ASPNETCORE_URLS=http://+:8080 \
    ASPNETCORE_FORWARDEDHEADERS_ENABLED=true \
    DOTNET_RUNNING_IN_CONTAINER=true
USER $APP_UID
ENTRYPOINT ["dotnet", "ui.dll"]
