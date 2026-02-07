# Stage 1: Сборка SDK и тестового сервиса
FROM harbor.kryukov.lan/homelab/maven:3.9-eclipse-temurin-21 AS builder

WORKDIR /workspace

# Копируем POM-файлы для кэширования зависимостей
COPY sdk-java/pom.xml sdk-java/pom.xml
COPY sdk-java/dephealth-core/pom.xml sdk-java/dephealth-core/pom.xml
COPY sdk-java/dephealth-spring-boot-starter/pom.xml sdk-java/dephealth-spring-boot-starter/pom.xml
COPY test-services/java-service/pom.xml test-services/java-service/pom.xml

# Загружаем зависимости
RUN cd sdk-java && mvn dependency:go-offline -q || true
RUN cd test-services/java-service && mvn dependency:go-offline -q || true

# Копируем исходники SDK
COPY sdk-java/dephealth-core/src sdk-java/dephealth-core/src
COPY sdk-java/dephealth-spring-boot-starter/src sdk-java/dephealth-spring-boot-starter/src

# Собираем и устанавливаем SDK в локальный Maven
RUN cd sdk-java && mvn install -DskipTests -q

# Копируем исходники тестового сервиса
COPY test-services/java-service/src test-services/java-service/src

# Собираем тестовый сервис
RUN cd test-services/java-service && mvn package -DskipTests -q

# Stage 2: Runtime
FROM harbor.kryukov.lan/homelab/eclipse-temurin:21-jre-alpine

WORKDIR /app

COPY --from=builder /workspace/test-services/java-service/target/*.jar app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
