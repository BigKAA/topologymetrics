# Stage 1: Сборка SDK и conformance-сервиса
FROM harbor.kryukov.lan/homelab/maven:3.9-eclipse-temurin-21 AS builder

WORKDIR /workspace

# Копируем POM-файлы для кэширования зависимостей
COPY sdk-java/pom.xml sdk-java/pom.xml
COPY sdk-java/dephealth-core/pom.xml sdk-java/dephealth-core/pom.xml
COPY sdk-java/dephealth-spring-boot-starter/pom.xml sdk-java/dephealth-spring-boot-starter/pom.xml
COPY conformance/test-service-java/pom.xml conformance/test-service-java/pom.xml

# Загружаем зависимости
RUN cd sdk-java && mvn dependency:go-offline -q || true
RUN cd conformance/test-service-java && mvn dependency:go-offline -q || true

# Копируем исходники SDK
COPY sdk-java/dephealth-core/src sdk-java/dephealth-core/src
COPY sdk-java/dephealth-spring-boot-starter/src sdk-java/dephealth-spring-boot-starter/src

# Собираем и устанавливаем SDK в локальный Maven
RUN cd sdk-java && mvn install -DskipTests -q

# Копируем исходники conformance-сервиса
COPY conformance/test-service-java/src conformance/test-service-java/src

# Собираем conformance-сервис
RUN cd conformance/test-service-java && mvn package -DskipTests -q

# Stage 2: Runtime
FROM harbor.kryukov.lan/homelab/eclipse-temurin:21-jre-alpine

RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /workspace/conformance/test-service-java/target/*.jar app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
