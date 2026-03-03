package store_test

import (
	"testing"
	"time"

	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/testutil"
)

func TestStore_ConversationLifecycle(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	// Create conversation
	conv := &store.Conversation{
		Title:        "Test Conversation",
		Model:        "kimi-k2.5",
		SystemPrompt: "You are a helpful assistant",
	}

	t.Run("CreateConversation", func(t *testing.T) {
		err := st.CreateConversation(conv)
		if err != nil {
			t.Fatalf("Failed to create conversation: %v", err)
		}
		if conv.ID == "" {
			t.Error("Conversation ID should be auto-generated")
		}
	})

	t.Run("GetConversation", func(t *testing.T) {
		retrieved, err := st.GetConversation(conv.ID)
		if err != nil {
			t.Fatalf("Failed to get conversation: %v", err)
		}
		if retrieved.Title != conv.Title {
			t.Errorf("Expected title %q, got %q", conv.Title, retrieved.Title)
		}
		if retrieved.Model != conv.Model {
			t.Errorf("Expected model %q, got %q", conv.Model, retrieved.Model)
		}
	})

	t.Run("UpdateConversation", func(t *testing.T) {
		conv.Title = "Updated Title"
		err := st.UpdateConversation(conv)
		if err != nil {
			t.Fatalf("Failed to update conversation: %v", err)
		}

		retrieved, err := st.GetConversation(conv.ID)
		if err != nil {
			t.Fatalf("Failed to get updated conversation: %v", err)
		}
		if retrieved.Title != "Updated Title" {
			t.Errorf("Expected title %q, got %q", "Updated Title", retrieved.Title)
		}
	})

	t.Run("ListConversations", func(t *testing.T) {
		convs, err := st.ListConversations(10, 0)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}
		if len(convs) == 0 {
			t.Error("Expected at least one conversation")
		}
	})

	t.Run("DeleteConversation", func(t *testing.T) {
		err := st.DeleteConversation(conv.ID)
		if err != nil {
			t.Fatalf("Failed to delete conversation: %v", err)
		}

		retrieved, err := st.GetConversation(conv.ID)
		if err != nil {
			t.Fatalf("Failed to get conversation after delete: %v", err)
		}
		if !retrieved.IsArchived {
			t.Error("Conversation should be archived after delete")
		}
	})
}

func TestStore_MessageOperations(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	// Create a conversation first
	conv := &store.Conversation{Title: "Message Test"}
	if err := st.CreateConversation(conv); err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	t.Run("CreateMessages", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			msg := &store.Message{
				ConversationID: conv.ID,
				Role:           "user",
				Content:        "Message " + string(rune('0'+i)),
				Tokens:         10,
			}
			if err := st.CreateMessage(msg); err != nil {
				t.Fatalf("Failed to create message %d: %v", i, err)
			}
		}
	})

	t.Run("GetMessages", func(t *testing.T) {
		msgs, err := st.GetMessages(conv.ID, 10, 0)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}
		if len(msgs) != 5 {
			t.Errorf("Expected 5 messages, got %d", len(msgs))
		}
	})

	t.Run("GetMessageCount", func(t *testing.T) {
		count, err := st.GetMessageCount(conv.ID)
		if err != nil {
			t.Fatalf("Failed to get message count: %v", err)
		}
		if count != 5 {
			t.Errorf("Expected count 5, got %d", count)
		}
	})

	t.Run("MessagePagination", func(t *testing.T) {
		// Test limit
		msgs, err := st.GetMessages(conv.ID, 2, 0)
		if err != nil {
			t.Fatalf("Failed to get paginated messages: %v", err)
		}
		if len(msgs) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(msgs))
		}

		// Test offset
		msgs, err = st.GetMessages(conv.ID, 2, 2)
		if err != nil {
			t.Fatalf("Failed to get offset messages: %v", err)
		}
		if len(msgs) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(msgs))
		}
	})

	t.Run("MessageWithToolCalls", func(t *testing.T) {
		// Create message with tool calls (JSON field)
		toolCallsJSON := store.JSON(`[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]`)
		msg := &store.Message{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        "",
			ToolCalls:      toolCallsJSON,
			Tokens:         50,
		}
		if err := st.CreateMessage(msg); err != nil {
			t.Fatalf("Failed to create message with tool calls: %v", err)
		}

		// Retrieve and verify
		msgs, err := st.GetMessages(conv.ID, 1, 0)
		if err != nil {
			t.Fatalf("Failed to get messages with tool calls: %v", err)
		}

		found := false
		for _, m := range msgs {
			if m.ID == msg.ID {
				found = true
				if len(m.ToolCalls) == 0 {
					t.Error("Expected ToolCalls to be retrieved, got empty")
				}
				// Verify the content matches
				if string(m.ToolCalls) != string(toolCallsJSON) {
					t.Errorf("ToolCalls mismatch: expected %s, got %s", toolCallsJSON, m.ToolCalls)
				}
				break
			}
		}
		if !found {
			t.Error("Message with tool calls not found in retrieved messages")
		}
	})

	t.Run("MessageWithToolResults", func(t *testing.T) {
		// Create message with tool results (JSON field)
		toolResultsJSON := store.JSON(`{"result":"sunny","temperature":25}`)
		msg := &store.Message{
			ConversationID: conv.ID,
			Role:           "tool",
			Content:        "The weather is sunny with 25°C",
			ToolResults:    toolResultsJSON,
			ToolCallID:     "call_1",
			Tokens:         20,
		}
		if err := st.CreateMessage(msg); err != nil {
			t.Fatalf("Failed to create message with tool results: %v", err)
		}

		// Retrieve and verify
		msgs, err := st.GetMessages(conv.ID, 1, 0)
		if err != nil {
			t.Fatalf("Failed to get messages with tool results: %v", err)
		}

		found := false
		for _, m := range msgs {
			if m.ID == msg.ID {
				found = true
				if len(m.ToolResults) == 0 {
					t.Error("Expected ToolResults to be retrieved, got empty")
				}
				break
			}
		}
		if !found {
			t.Error("Message with tool results not found in retrieved messages")
		}
	})
}

func TestStore_MemoryOperations(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	t.Run("CreateMemory", func(t *testing.T) {
		mem := &store.Memory{
			Type:       "fact",
			Content:    "User prefers dark mode",
			Importance: 8,
			Source:     "conversation_123",
		}

		err := st.CreateMemory(mem)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}
		if mem.ID == "" {
			t.Error("Memory ID should be auto-generated")
		}
		if mem.Importance != 8 {
			t.Errorf("Expected importance 8, got %d", mem.Importance)
		}
	})

	t.Run("CreateMemory_Defaults", func(t *testing.T) {
		mem := &store.Memory{
			Content: "Simple fact without type or importance",
		}

		err := st.CreateMemory(mem)
		if err != nil {
			t.Fatalf("Failed to create memory with defaults: %v", err)
		}
		if mem.Type != "fact" {
			t.Errorf("Expected default type 'fact', got %q", mem.Type)
		}
		if mem.Importance != 5 {
			t.Errorf("Expected default importance 5, got %d", mem.Importance)
		}
	})

	t.Run("SearchMemories", func(t *testing.T) {
		// Create test memories
		memories := []string{
			"User likes golang programming",
			"User prefers morning meetings",
			"User has a dog named Max",
		}

		for _, content := range memories {
			mem := &store.Memory{Content: content}
			if err := st.CreateMemory(mem); err != nil {
				t.Fatalf("Failed to create memory: %v", err)
			}
		}

		// Search for "golang"
		results, err := st.SearchMemories("golang", 10)
		if err != nil {
			t.Fatalf("Failed to search memories: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected to find memory containing 'golang'")
		}
		found := false
		for _, mem := range results {
			if mem.Content == "User likes golang programming" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Did not find expected memory in search results")
		}
	})

	t.Run("GetRecentMemories", func(t *testing.T) {
		results, err := st.GetRecentMemories(5)
		if err != nil {
			t.Fatalf("Failed to get recent memories: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected to find recent memories")
		}
	})
}

func TestStore_ChatMapping(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	chatID := int64(123456)
	chatType := "telegram"
	convID := "conv_test_123"

	t.Run("SetChatMapping", func(t *testing.T) {
		err := st.SetChatMapping(chatID, chatType, convID)
		if err != nil {
			t.Fatalf("Failed to set chat mapping: %v", err)
		}
	})

	t.Run("GetChatMapping", func(t *testing.T) {
		mapping, err := st.GetChatMapping(chatID, chatType)
		if err != nil {
			t.Fatalf("Failed to get chat mapping: %v", err)
		}
		if mapping.ConversationID != convID {
			t.Errorf("Expected conversation ID %q, got %q", convID, mapping.ConversationID)
		}
		if mapping.ChatID != chatID {
			t.Errorf("Expected chat ID %d, got %d", chatID, mapping.ChatID)
		}
	})

	t.Run("UpdateChatMapping", func(t *testing.T) {
		newConvID := "conv_test_456"
		err := st.SetChatMapping(chatID, chatType, newConvID)
		if err != nil {
			t.Fatalf("Failed to update chat mapping: %v", err)
		}

		mapping, err := st.GetChatMapping(chatID, chatType)
		if err != nil {
			t.Fatalf("Failed to get updated mapping: %v", err)
		}
		if mapping.ConversationID != newConvID {
			t.Errorf("Expected updated conversation ID %q, got %q", newConvID, mapping.ConversationID)
		}
	})

	t.Run("GetChatConversationHistory", func(t *testing.T) {
		history, err := st.GetChatConversationHistory(chatID, chatType, 10)
		if err != nil {
			t.Fatalf("Failed to get chat history: %v", err)
		}
		// Should have both old and new mappings
		if len(history) < 1 {
			t.Errorf("Expected at least 1 mapping in history, got %d", len(history))
		}
	})

	t.Run("DeactivateChatMapping", func(t *testing.T) {
		err := st.DeactivateChatMapping(chatID, chatType)
		if err != nil {
			t.Fatalf("Failed to deactivate chat mapping: %v", err)
		}

		// After deactivation, GetChatMapping should fail
		_, err = st.GetChatMapping(chatID, chatType)
		if err == nil {
			t.Error("Expected error when getting deactivated mapping")
		}
	})
}

func TestStore_FileOperations(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	convID := "conv_file_test"
	conv := &store.Conversation{ID: convID, Title: "File Test"}
	st.CreateConversation(conv)

	t.Run("CreateFile", func(t *testing.T) {
		file := &store.File{
			Filename:       "test.txt",
			MimeType:       "text/plain",
			SizeBytes:      1024,
			StoragePath:    "/tmp/test.txt",
			ConversationID: &convID,
		}

		err := st.CreateFile(file)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if file.ID == "" {
			t.Error("File ID should be auto-generated")
		}
	})

	t.Run("GetFile", func(t *testing.T) {
		// Create a file first
		file := &store.File{
			Filename:    "get_test.txt",
			MimeType:    "text/plain",
			SizeBytes:   2048,
			StoragePath: "/tmp/get_test.txt",
		}
		if err := st.CreateFile(file); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		retrieved, err := st.GetFile(file.ID)
		if err != nil {
			t.Fatalf("Failed to get file: %v", err)
		}
		if retrieved.Filename != file.Filename {
			t.Errorf("Expected filename %q, got %q", file.Filename, retrieved.Filename)
		}
	})

	t.Run("ListFiles", func(t *testing.T) {
		files, err := st.ListFiles(&convID, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}
		if len(files) == 0 {
			t.Error("Expected to find files for conversation")
		}
	})

	t.Run("UpdateFileProcessedText", func(t *testing.T) {
		file := &store.File{
			Filename:    "process_test.txt",
			MimeType:    "text/plain",
			SizeBytes:   512,
			StoragePath: "/tmp/process_test.txt",
		}
		if err := st.CreateFile(file); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		processedText := "This is the extracted text from the file"
		err := st.UpdateFileProcessedText(file.ID, processedText)
		if err != nil {
			t.Fatalf("Failed to update processed text: %v", err)
		}

		retrieved, err := st.GetFile(file.ID)
		if err != nil {
			t.Fatalf("Failed to get updated file: %v", err)
		}
		if retrieved.ProcessedText != processedText {
			t.Errorf("Expected processed text %q, got %q", processedText, retrieved.ProcessedText)
		}
	})
}

func TestStore_TaskOperations(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	t.Run("CreateTask", func(t *testing.T) {
		task := &store.Task{
			Type:   "analysis",
			Status: "pending",
			Title:  "Test Analysis Task",
			Prompt: "Analyze the following data...",
		}

		err := st.CreateTask(task)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		if task.ID == "" {
			t.Error("Task ID should be auto-generated")
		}
		if task.Status != "pending" {
			t.Errorf("Expected status 'pending', got %q", task.Status)
		}
	})

	t.Run("GetPendingTasks", func(t *testing.T) {
		// Create several tasks
		for i := 0; i < 3; i++ {
			task := &store.Task{
				Type:   "background",
				Status: "pending",
				Title:  "Pending Task",
				Prompt: "Process data...",
			}
			st.CreateTask(task)
		}

		tasks, err := st.GetPendingTasks(10)
		if err != nil {
			t.Fatalf("Failed to get pending tasks: %v", err)
		}
		if len(tasks) == 0 {
			t.Error("Expected to find pending tasks")
		}
	})

	t.Run("UpdateTask", func(t *testing.T) {
		task := &store.Task{
			Type:   "update_test",
			Status: "pending",
			Title:  "Update Test",
			Prompt: "Test prompt",
		}
		if err := st.CreateTask(task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		task.Status = "running"
		now := time.Now()
		task.StartedAt = &now
		err := st.UpdateTask(task)
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		// Verify by getting pending tasks (should not include this one)
		pending, _ := st.GetPendingTasks(100)
		for _, p := range pending {
			if p.ID == task.ID {
				t.Error("Updated task should not appear in pending list")
			}
		}
	})
}

func TestStore_ScheduledJobOperations(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	t.Run("CreateJob", func(t *testing.T) {
		job := &store.ScheduledJob{
			Name:           "Daily Report",
			CronExpression: "0 9 * * *",
			Prompt:         "Generate daily summary",
			IsActive:       true,
		}

		err := st.CreateJob(job)
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}
		if job.ID == "" {
			t.Error("Job ID should be auto-generated")
		}
	})

	t.Run("GetJob", func(t *testing.T) {
		job := &store.ScheduledJob{
			Name:           "Get Test Job",
			CronExpression: "*/5 * * * *",
			Prompt:         "Test prompt",
			IsActive:       true,
		}
		if err := st.CreateJob(job); err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		retrieved, err := st.GetJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}
		if retrieved.Name != job.Name {
			t.Errorf("Expected name %q, got %q", job.Name, retrieved.Name)
		}
	})

	t.Run("UpdateJob", func(t *testing.T) {
		job := &store.ScheduledJob{
			Name:           "Update Test Job",
			CronExpression: "0 * * * *",
			Prompt:         "Original prompt",
			IsActive:       true,
		}
		if err := st.CreateJob(job); err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		job.Prompt = "Updated prompt"
		job.RunCount = 1
		err := st.UpdateJob(job)
		if err != nil {
			t.Fatalf("Failed to update job: %v", err)
		}

		retrieved, err := st.GetJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to get updated job: %v", err)
		}
		if retrieved.Prompt != "Updated prompt" {
			t.Errorf("Expected prompt %q, got %q", "Updated prompt", retrieved.Prompt)
		}
		if retrieved.RunCount != 1 {
			t.Errorf("Expected run count 1, got %d", retrieved.RunCount)
		}
	})

	t.Run("ListJobs", func(t *testing.T) {
		jobs, err := st.ListJobs()
		if err != nil {
			t.Fatalf("Failed to list jobs: %v", err)
		}
		if len(jobs) == 0 {
			t.Error("Expected to find scheduled jobs")
		}
	})

	t.Run("DeleteJob", func(t *testing.T) {
		job := &store.ScheduledJob{
			Name:           "Delete Test Job",
			CronExpression: "0 0 * * *",
			Prompt:         "To be deleted",
			IsActive:       true,
		}
		if err := st.CreateJob(job); err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		err := st.DeleteJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to delete job: %v", err)
		}

		_, err = st.GetJob(job.ID)
		if err == nil {
			t.Error("Expected error when getting deleted job")
		}
	})

	t.Run("GetDueJobs", func(t *testing.T) {
		// Create a job with past next_run_at
		job := &store.ScheduledJob{
			Name:           "Due Job",
			CronExpression: "0 0 * * *",
			Prompt:         "Should run now",
			IsActive:       true,
		}
		pastTime := time.Now().Add(-1 * time.Hour)
		job.NextRunAt = &pastTime

		if err := st.CreateJob(job); err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		// Update to set the NextRunAt
		st.DB().Model(job).Update("next_run_at", pastTime)

		dueJobs, err := st.GetDueJobs(10)
		if err != nil {
			t.Fatalf("Failed to get due jobs: %v", err)
		}

		found := false
		for _, j := range dueJobs {
			if j.ID == job.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find due job in results")
		}
	})
}

func TestStore_BadgerOperations(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	t.Run("SessionOperations", func(t *testing.T) {
		key := "test_session"
		value := []byte("session_data_123")

		// Set session
		err := st.SetSession(key, value, 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to set session: %v", err)
		}

		// Get session
		retrieved, err := st.GetSession(key)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}
		if string(retrieved) != string(value) {
			t.Errorf("Expected %q, got %q", string(value), string(retrieved))
		}

		// Delete session
		err = st.DeleteSession(key)
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		// Verify deletion
		_, err = st.GetSession(key)
		if err == nil {
			t.Error("Expected error when getting deleted session")
		}
	})

	t.Run("KVOperations", func(t *testing.T) {
		key := "test_key"
		value := []byte("test_value")

		// Set KV
		err := st.SetKV(key, value)
		if err != nil {
			t.Fatalf("Failed to set KV: %v", err)
		}

		// Get KV
		retrieved, err := st.GetKV(key)
		if err != nil {
			t.Fatalf("Failed to get KV: %v", err)
		}
		if string(retrieved) != string(value) {
			t.Errorf("Expected %q, got %q", string(value), string(retrieved))
		}
	})

	t.Run("QueueOperations", func(t *testing.T) {
		queueName := "test_queue"
		job1 := []byte("job_1")
		job2 := []byte("job_2")

		// Enqueue jobs
		if err := st.Enqueue(queueName, job1); err != nil {
			t.Fatalf("Failed to enqueue job1: %v", err)
		}
		if err := st.Enqueue(queueName, job2); err != nil {
			t.Fatalf("Failed to enqueue job2: %v", err)
		}

		// Dequeue jobs (FIFO order)
		retrieved1, err := st.Dequeue(queueName)
		if err != nil {
			t.Fatalf("Failed to dequeue job1: %v", err)
		}
		if string(retrieved1) != string(job1) {
			t.Errorf("Expected %q, got %q", string(job1), string(retrieved1))
		}

		retrieved2, err := st.Dequeue(queueName)
		if err != nil {
			t.Fatalf("Failed to dequeue job2: %v", err)
		}
		if string(retrieved2) != string(job2) {
			t.Errorf("Expected %q, got %q", string(job2), string(retrieved2))
		}

		// Queue should be empty now
		_, err = st.Dequeue(queueName)
		if err == nil {
			t.Error("Expected error when dequeuing from empty queue")
		}
	})
}

func TestStore_ConcurrentAccess(t *testing.T) {
	st := testutil.NewTestStore(t)
	defer st.Close()

	// Test concurrent conversation creation
	t.Run("ConcurrentConversations", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				conv := &store.Conversation{
					Title: "Concurrent " + string(rune('0'+idx)),
				}
				err := st.CreateConversation(conv)
				if err != nil {
					t.Errorf("Failed to create conversation %d: %v", idx, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all were created
		convs, err := st.ListConversations(20, 0)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}
		if len(convs) < 10 {
			t.Errorf("Expected at least 10 conversations, got %d", len(convs))
		}
	})
}
