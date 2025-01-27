# OpenTelemetry Instrumentation Examples

Requirements:

- Go 1.20
- Docker Compose v2

Build and run the backend:

```sh
docker compose up -d
```

Run the **user-service** app

```sh
cd user-service
go run main.go
```

Run the **order-service** app

```sh
cd order-service
go run main.go
```

Run curl to see traces in action

```sh
curl http://localhost:8080/user
```

Browse exported telemetry:

- [Traces](http://localhost:16686)
- [Metrics](http://localhost:8889/metrics)

**Baggage showing up in user and order service tags**

![Screenshot 2025-01-21 at 1 02 42 AM](https://github.com/user-attachments/assets/12486aea-8bb8-43cc-ae12-aab7119f6c74)
