package generatedapp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecutionLogCollectorCapturesStreamsAndRespectsLimit(t *testing.T) {
	var observed []ExecutionLogEvent
	collector := newExecutionLogCollector(12, func(event ExecutionLogEvent) {
		observed = append(observed, event)
	})

	if _, err := collector.Writer("stdout").Write([]byte("hello")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if _, err := collector.Writer("stderr").Write([]byte(" world!!!")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	entries, truncated := collector.Entries()
	if !truncated {
		t.Fatal("expected log truncation")
	}
	if len(entries) != 2 || len(observed) != 2 {
		t.Fatalf("expected two emitted events, got entries=%d observed=%d", len(entries), len(observed))
	}
	if entries[0].Stream != "stdout" || entries[0].Content != "hello" {
		t.Fatalf("unexpected stdout event: %#v", entries[0])
	}
	if entries[1].Stream != "stderr" || entries[1].Content != " world!" {
		t.Fatalf("unexpected truncated stderr event: %#v", entries[1])
	}
	if entries[0].Timestamp == 0 || entries[1].Timestamp == 0 {
		t.Fatal("expected timestamps on log events")
	}
}

func TestExecutionLogCollectorSplitsWASIWritesIntoConsoleLines(t *testing.T) {
	collector := newExecutionLogCollector(1024, nil)
	writer := collector.Writer("stderr")
	if _, err := writer.Write([]byte("2026/08/24 created")); err != nil {
		t.Fatalf("write first chunk: %v", err)
	}
	if _, err := writer.Write([]byte(" book\n2026/08/24 updated book\nlast")); err != nil {
		t.Fatalf("write second chunk: %v", err)
	}

	entries, truncated := collector.Entries()
	if truncated {
		t.Fatal("did not expect truncation")
	}
	if len(entries) != 3 {
		t.Fatalf("expected three console lines, got %#v", entries)
	}
	for index, want := range []string{"2026/08/24 created book", "2026/08/24 updated book", "last"} {
		if entries[index].Content != want || entries[index].Stream != "stderr" {
			t.Fatalf("unexpected console event %d: %#v", index, entries[index])
		}
	}
}

func TestRuntimeLogsRecordActionAndGuestResultWithoutGuestLogCalls(t *testing.T) {
	collector := newExecutionLogCollector(1024, nil)
	ctx := context.WithValue(context.Background(), executionLogCollectorContextKey{}, collector)
	ctx = context.WithValue(ctx, invocationActionContextKey{}, "book_create")

	emitRuntimeLog(ctx, "INFO action=book_create received")
	(&Service{}).logGuestResult(ctx, []byte(`{"success":true}`))

	entries, truncated := collector.Entries()
	if truncated || len(entries) != 2 {
		t.Fatalf("runtime log entries = %#v, truncated=%v", entries, truncated)
	}
	if entries[0].Content != "INFO action=book_create received" || entries[1].Content != "INFO action=book_create result=success" {
		t.Fatalf("unexpected runtime logs: %#v", entries)
	}
}

func TestDataOperationLogsAreCorrelatedAndDoNotExposeRequestValues(t *testing.T) {
	collector := newExecutionLogCollector(1024, nil)
	ctx := context.WithValue(context.Background(), executionLogCollectorContextKey{}, collector)
	ctx = context.WithValue(ctx, invocationActionContextKey{}, "book_create")
	startedAt := logDataOperationStart(ctx, "data_create", "model", "book")
	logDataOperationSuccess(ctx, "data_create", "model", "book", startedAt, "rows_affected=1")
	logDataOperationFailure(ctx, "data_update", "model", "book", time.Now(), "data_update_failed")

	entries, truncated := collector.Entries()
	if truncated || len(entries) != 3 {
		t.Fatalf("data operation logs = %#v, truncated=%v", entries, truncated)
	}
	contents := make([]string, 0, len(entries))
	for _, entry := range entries {
		contents = append(contents, entry.Content)
	}
	joined := strings.Join(contents, "\n")
	for _, want := range []string{
		"INFO action=book_create operation=data_create model=book phase=start",
		"INFO action=book_create operation=data_create model=book result=success rows_affected=1 duration_ms=",
		"ERROR action=book_create operation=data_update model=book result=failed code=data_update_failed duration_ms=",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing log %q in %#v", want, contents)
		}
	}
	if strings.Contains(joined, "title=private") {
		t.Fatalf("raw request value leaked into data operation logs: %#v", contents)
	}
}

func TestInvocationAction(t *testing.T) {
	if got := invocationAction([]byte(`{"action":"book_list"}`)); got != "book_list" {
		t.Fatalf("invocation action = %q", got)
	}
	if got := invocationAction([]byte(`{}`)); got != "handle" {
		t.Fatalf("fallback action = %q", got)
	}
}

func TestIsWASMErrorLog(t *testing.T) {
	if isWASMErrorLog("stderr", "2026/08/24 12:00:00 [INFO] processed order") {
		t.Fatal("standard logger info should not be promoted to error")
	}
	if !isWASMErrorLog("stderr", "[ERROR] database write failed") {
		t.Fatal("explicit error logs should use error level")
	}
	if !isWASMErrorLog("stderr", "panic: invalid memory address") {
		t.Fatal("panics should use error level")
	}
}
