// main.go - Service A (User Service)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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
	tracer = otel.Tracer("user-service")
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create baggage with user type
	userTypeMember, _ := baggage.NewMember("user.type", "premium")
	bag, _ := baggage.New(userTypeMember)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	// Start a span
	ctx, span := tracer.Start(ctx, "get-user")
	defer span.End()

	// Add baggage as span attributes for visibility
	for _, member := range bag.Members() {
		span.SetAttributes(attribute.String("baggage."+member.Key(), member.Value()))
	}

	// Get user from database
	user := getUser(ctx, 1)

	// Call order service
	getOrders(ctx, user.ID)

	json.NewEncoder(w).Encode(user)
}

func getUser(ctx context.Context, id int) User {
	// Extract baggage to show it's available
	bag := baggage.FromContext(ctx)
	userType := bag.Member("user.type").Value()
	log.Printf("Processing request for user type: %s", userType)

	var user User
	err := db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		log.Printf("Error querying user: %v", err)
		return User{}
	}
	return user
}

func getOrders(ctx context.Context, userID int) {
	// Make HTTP request to order service
	client := &http.Client{}

	// Convert userID to string properly using strconv
	userIDStr := strconv.Itoa(userID)

	// Handle the error from NewRequestWithContext
	req, err := http.NewRequestWithContext(ctx, "GET",
		"http://localhost:8081/orders?user_id="+userIDStr, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling order service: %v", err)
		return
	}
	defer resp.Body.Close()
}

func main() {
	// Connect to database
	var dbErr error
	// Wrap sql.Open with OpenTelemetry instrumentation
	db, dbErr = otelsql.Open("postgres", "postgres://postgres:pswd@localhost:5433/postgres?sslmode=disable",
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if dbErr != nil {
		log.Fatal(dbErr)
	}

	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	shutdown, err := telLib.Run(context.Background(), "user-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run OpenTelemetry: %v\n", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shutdown OpenTelemetry: %v\n", err)
		}
	}()

	// Start HTTP server
	http.HandleFunc("/user", getUserHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
