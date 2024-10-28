// registry_test.go
package registry

import (
    "context"
    "testing"
    "time"
)

func TestRegistryBasicOperations(t *testing.T) {
    // Initialize registry with test configuration
    registry := NewRegistry[string, string](
        WithTTL(time.Hour),
        WithMaxItems(100),
    )

    // Test context
    ctx := context.Background()

    // Test cases
    tests := []struct {
        name    string
        key     string
        value   string
        wantErr bool
    }{
        {
            name:    "Set and get basic item",
            key:     "test-key",
            value:   "test-value",
            wantErr: false,
        },
        {
            name:    "Get non-existent item",
            key:     "non-existent",
            value:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test Set operation
            if tt.value != "" {
                err := registry.Set(ctx, tt.key, tt.value)
                if err != nil && !tt.wantErr {
                    t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
                    return
                }
            }

            // Test Get operation
            got, err := registry.Get(ctx, tt.key)
            if (err != nil) != tt.wantErr {
                t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            // Verify value if no error expected
            if !tt.wantErr && got.Value != tt.value {
                t.Errorf("Get() got = %v, want %v", got.Value, tt.value)
            }
        })
    }
}

func TestRegistryEventSubscription(t *testing.T) {
    registry := NewRegistry[string, string]()
    ctx := context.Background()

    // Create channel to receive events
    events := make(chan Event[string], 1)
    
    // Subscribe to registry events
    registry.Subscribe(func(e Event[string]) {
        events <- e
    })

    // Set a value to trigger an event
    testKey := "event-test"
    testValue := "test-value"
    
    err := registry.Set(ctx, testKey, testValue)
    if err != nil {
        t.Fatalf("Failed to set value: %v", err)
    }

    // Wait for event with timeout
    select {
    case event := <-events:
        if event.Type != EventCreated {
            t.Errorf("Expected event type %v, got %v", EventCreated, event.Type)
        }
        if event.Key != testKey {
            t.Errorf("Expected key %v, got %v", testKey, event.Key)
        }
    case <-time.After(time.Second):
        t.Error("Timeout waiting for event")
    }
}