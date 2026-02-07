# Build context: корень проекта (dephealth/)
# docker build -t ... -f test-services/csharp-service/Dockerfile .

# Stage 1: Сборка SDK и тестового сервиса
ARG MCR_REGISTRY=mcr.microsoft.com
FROM ${MCR_REGISTRY}/dotnet/sdk:8.0 AS builder

WORKDIR /workspace

# Копируем файлы проектов для кэширования restore
COPY sdk-csharp/Directory.Build.props sdk-csharp/
COPY sdk-csharp/DepHealth.Core/DepHealth.Core.csproj sdk-csharp/DepHealth.Core/
COPY sdk-csharp/DepHealth.AspNetCore/DepHealth.AspNetCore.csproj sdk-csharp/DepHealth.AspNetCore/
COPY sdk-csharp/DepHealth.EntityFramework/DepHealth.EntityFramework.csproj sdk-csharp/DepHealth.EntityFramework/
COPY test-services/csharp-service/csharp-service.csproj test-services/csharp-service/

# Restore зависимостей
RUN dotnet restore test-services/csharp-service/csharp-service.csproj

# Копируем исходники SDK
COPY sdk-csharp/DepHealth.Core/ sdk-csharp/DepHealth.Core/
COPY sdk-csharp/DepHealth.AspNetCore/ sdk-csharp/DepHealth.AspNetCore/
COPY sdk-csharp/DepHealth.EntityFramework/ sdk-csharp/DepHealth.EntityFramework/

# Копируем исходники тестового сервиса
COPY test-services/csharp-service/ test-services/csharp-service/

# Publish
RUN dotnet publish test-services/csharp-service/csharp-service.csproj \
    -c Release -o /app/publish --no-restore

# Stage 2: Runtime
ARG MCR_REGISTRY=mcr.microsoft.com
FROM ${MCR_REGISTRY}/dotnet/aspnet:8.0-alpine

WORKDIR /app

COPY --from=builder /app/publish .

EXPOSE 8080

ENTRYPOINT ["dotnet", "csharp-service.dll"]
