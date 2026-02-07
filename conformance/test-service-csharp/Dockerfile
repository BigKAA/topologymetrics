# Build context: корень проекта (dephealth/)
# docker build -t ... -f conformance/test-service-csharp/Dockerfile .

# Stage 1: Сборка SDK и conformance-сервиса
ARG MCR_REGISTRY=mcr.microsoft.com
FROM ${MCR_REGISTRY}/dotnet/sdk:8.0 AS builder

WORKDIR /workspace

# Копируем файлы проектов для кэширования restore
COPY sdk-csharp/Directory.Build.props sdk-csharp/
COPY sdk-csharp/DepHealth.Core/DepHealth.Core.csproj sdk-csharp/DepHealth.Core/
COPY sdk-csharp/DepHealth.AspNetCore/DepHealth.AspNetCore.csproj sdk-csharp/DepHealth.AspNetCore/
COPY sdk-csharp/DepHealth.EntityFramework/DepHealth.EntityFramework.csproj sdk-csharp/DepHealth.EntityFramework/
COPY conformance/test-service-csharp/conformance-service.csproj conformance/test-service-csharp/

# Restore зависимостей
RUN dotnet restore conformance/test-service-csharp/conformance-service.csproj

# Копируем исходники SDK
COPY sdk-csharp/DepHealth.Core/ sdk-csharp/DepHealth.Core/
COPY sdk-csharp/DepHealth.AspNetCore/ sdk-csharp/DepHealth.AspNetCore/
COPY sdk-csharp/DepHealth.EntityFramework/ sdk-csharp/DepHealth.EntityFramework/

# Копируем исходники conformance-сервиса
COPY conformance/test-service-csharp/ conformance/test-service-csharp/

# Publish
RUN dotnet publish conformance/test-service-csharp/conformance-service.csproj \
    -c Release -o /app/publish --no-restore

# Stage 2: Runtime
ARG MCR_REGISTRY=mcr.microsoft.com
FROM ${MCR_REGISTRY}/dotnet/aspnet:8.0-alpine

# curl нужен для timeout.yml сценария (HTTP PUT из pod)
RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /app/publish .

EXPOSE 8080

ENTRYPOINT ["dotnet", "conformance-service.dll"]
