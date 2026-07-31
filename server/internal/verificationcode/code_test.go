package verificationcode

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestVerifyDoesNotConsumeAndConsumeDeletes(t *testing.T) {
	ctx := context.Background()
	client := verificationCodeTestClient(t)
	if err := client.Set(ctx, "code", `{"code":"123456","lastAt":1}`, 0).Err(); err != nil {
		t.Fatal(err)
	}
	if ok, err := Verify(ctx, client, "code", "123456"); err != nil || !ok {
		t.Fatalf("Verify() = %v, %v", ok, err)
	}
	if exists := client.Exists(ctx, "code").Val(); exists != 1 {
		t.Fatalf("key exists after verify = %d, want 1", exists)
	}
	if ok, err := Consume(ctx, client, "code", "123456"); err != nil || !ok {
		t.Fatalf("Consume() = %v, %v", ok, err)
	}
	if exists := client.Exists(ctx, "code").Val(); exists != 0 {
		t.Fatalf("key exists after consume = %d, want 0", exists)
	}
}

func TestConsumeKeepsValueOnMismatchAndSupportsLegacyRawCode(t *testing.T) {
	ctx := context.Background()
	client := verificationCodeTestClient(t)
	if err := client.Set(ctx, "code", "654321", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if ok, err := Consume(ctx, client, "code", "wrong"); err != nil || ok {
		t.Fatalf("mismatched Consume() = %v, %v", ok, err)
	}
	if ok, err := Consume(ctx, client, "code", "654321"); err != nil || !ok {
		t.Fatalf("legacy Consume() = %v, %v", ok, err)
	}
}

func TestVerificationCodeIsInvalidatedAfterRepeatedMismatches(t *testing.T) {
	ctx := context.Background()
	client := verificationCodeTestClient(t)
	if err := client.Set(ctx, "code", "654321", 0).Err(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxVerificationAttempts; i++ {
		if ok, err := Verify(ctx, client, "code", "000000"); err != nil || ok {
			t.Fatalf("mismatched Verify() = %v, %v", ok, err)
		}
	}
	if exists := client.Exists(ctx, "code", attemptKey("code")).Val(); exists != 0 {
		t.Fatalf("verification state exists after attempt limit = %d, want 0", exists)
	}
	if ok, err := Consume(ctx, client, "code", "654321"); err != nil || ok {
		t.Fatalf("invalidated code Consume() = %v, %v", ok, err)
	}
}

func TestConcurrentConsumeSucceedsOnce(t *testing.T) {
	ctx := context.Background()
	client := verificationCodeTestClient(t)
	if err := client.Set(ctx, "code", `{"code":"123456"}`, 0).Err(); err != nil {
		t.Fatal(err)
	}
	const callers = 24
	start := make(chan struct{})
	var successes atomic.Int32
	var wg sync.WaitGroup
	wg.Add(callers)
	for range callers {
		go func() {
			defer wg.Done()
			<-start
			ok, err := Consume(ctx, client, "code", "123456")
			if err != nil {
				t.Errorf("Consume() error = %v", err)
				return
			}
			if ok {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("successful consumers = %d, want 1", got)
	}
}

func verificationCodeTestClient(t *testing.T) *redis.Client {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}
