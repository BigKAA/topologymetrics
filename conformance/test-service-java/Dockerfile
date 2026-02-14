# Stage 1: Сборка SDK и conformance-сервиса
ARG REGISTRY=docker.io
FROM ${REGISTRY}/maven:3.9-eclipse-temurin-21 AS builder

WORKDIR /workspace

# Download SDK dependencies and install SDK first
COPY sdk-java/pom.xml sdk-java/pom.xml
COPY sdk-java/dephealth-core/pom.xml sdk-java/dephealth-core/pom.xml
COPY sdk-java/dephealth-spring-boot-starter/pom.xml sdk-java/dephealth-spring-boot-starter/pom.xml

RUN cd sdk-java && mvn dependency:go-offline -q || true

COPY sdk-java/dephealth-core/src sdk-java/dephealth-core/src
COPY sdk-java/dephealth-spring-boot-starter/src sdk-java/dephealth-spring-boot-starter/src

RUN cd sdk-java && mvn install -DskipTests -q

# Build conformance service (SDK is already in local Maven repo)
COPY conformance/test-service-java/pom.xml conformance/test-service-java/pom.xml
COPY conformance/test-service-java/src conformance/test-service-java/src
RUN cd conformance/test-service-java && mvn package -DskipTests -q

# Stage 2: Runtime
ARG REGISTRY=docker.io
FROM ${REGISTRY}/eclipse-temurin:21-jre-alpine

RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /workspace/conformance/test-service-java/target/*.jar app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
