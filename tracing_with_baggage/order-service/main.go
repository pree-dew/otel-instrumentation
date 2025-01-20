// orders.go - Service B (Order Service)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	telLib "sampleapp/telemetry"
)

var (
	db     *sql.DB
	tracer = otel.Tracer("order-service")
)

type Order struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
}

func getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	// Extract context from headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// Start a span
	ctx, span := tracer.Start(ctx, "get-orders")
	defer span.End()

	// Extract baggage to show it's been propagated
	bag := baggage.FromContext(ctx)
	for _, member := range bag.Members() {
		span.SetAttributes(attribute.String("baggage."+member.Key(), member.Value()))
	}

	// Get orders from database
	userID := 1 // In real app, get from query params
	orders := getOrders(ctx, userID)

	json.NewEncoder(w).Encode(orders)
}

func getOrders(ctx context.Context, userID int) []Order {
	var orders []Order
	rows, err := db.QueryContext(ctx,
		"SELECT id, user_id, amount FROM orders WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Amount); err != nil {
			log.Printf("Error scanning order: %v", err)
			continue
		}
		orders = append(orders, order)
	}
	return orders
}

func main() {
	// Connect to database
	var dbErr error
	// Wrap sql.Open with OpenTelemetry instrumentation
	db, dbErr = otelsql.Open("postgres", "postgres://postgres:pswd@localhost:5433/postgres?sslmode=disable",
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
	if dbErr != nil {
		log.Fatal(dbErr)
	}

	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	shutdown, err := telLib.Run(context.Background(), "order-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run OpenTelemetry: %v\n", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shutdown OpenTelemetry: %v\n", err)
		}
	}()

	http.HandleFunc("/orders", getOrdersHandler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
