// Ported from: packages/core/src/storage/domains/memory/inmemory.ts
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ MemoryStorage = (*InMemoryMemory)(nil)

// ---------------------------------------------------------------------------
// InMemoryMemory
// ---------------------------------------------------------------------------

// InMemoryMemory is an in-memory implementation of MemoryStorage.
type InMemoryMemory struct {
	db *domains.InMemoryDB
}

// NewInMemoryMemory creates a new InMemoryMemory.
func NewInMemoryMemory(db *domains.InMemoryDB) *InMemoryMemory {
	return &InMemoryMemory{db: db}
}

// Init is a no-op for in-memory storage.
func (s *InMemoryMemory) Init(_ context.Context) error {
	return nil
}

// SupportsObservationalMemory returns true for the in-memory implementation.
func (s *InMemoryMemory) SupportsObservationalMemory() bool {
	return true
}

// DangerouslyClearAll clears all memory data.
func (s *InMemoryMemory) DangerouslyClearAll(_ context.Context) error {
	s.db.Lock()
	defer s.db.Unlock()

	s.db.Threads = make(map[string]any)
	s.db.Messages = make(map[string]any)
	s.db.Resources = make(map[string]any)
	s.db.ObservationalMemory = make(map[string][]any)
	return nil
}

// ---------------------------------------------------------------------------
// Thread Methods
// ---------------------------------------------------------------------------

// GetThreadByID retrieves a thread by its ID.
func (s *InMemoryMemory) GetThreadByID(_ context.Context, threadID string) (StorageThreadType, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	thread, ok := s.db.Threads[threadID]
	if !ok || thread == nil {
		return nil, nil
	}
	threadMap, ok := thread.(map[string]any)
	if !ok {
		return nil, nil
	}
	return cloneThread(threadMap), nil
}

// SaveThread creates or saves a thread.
func (s *InMemoryMemory) SaveThread(_ context.Context, thread StorageThreadType) (StorageThreadType, error) {
	s.db.Lock()
	defer s.db.Unlock()

	id, _ := thread["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("thread id is required")
	}
	s.db.Threads[id] = thread
	return thread, nil
}

// UpdateThread updates an existing thread.
func (s *InMemoryMemory) UpdateThread(_ context.Context, input UpdateThreadInput) (StorageThreadType, error) {
	s.db.Lock()
	defer s.db.Unlock()

	raw, ok := s.db.Threads[input.ID]
	if !ok || raw == nil {
		return nil, fmt.Errorf("thread with id %s not found", input.ID)
	}
	thread, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("thread with id %s not found", input.ID)
	}

	thread["title"] = input.Title
	// Merge metadata.
	existing, _ := thread["metadata"].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	for k, v := range input.Metadata {
		existing[k] = v
	}
	thread["metadata"] = existing
	thread["updatedAt"] = time.Now()

	return thread, nil
}

// DeleteThread removes a thread by ID and its messages.
func (s *InMemoryMemory) DeleteThread(_ context.Context, threadID string) error {
	s.db.Lock()
	defer s.db.Unlock()

	delete(s.db.Threads, threadID)

	// Delete associated messages.
	for key, raw := range s.db.Messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if msg["thread_id"] == threadID {
			delete(s.db.Messages, key)
		}
	}
	return nil
}

// ListThreads lists threads with optional filtering.
func (s *InMemoryMemory) ListThreads(_ context.Context, args StorageListThreadsInput) (StorageListThreadsOutput, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	field, direction := parseOrderBy(args.OrderBy, "DESC")

	perPage := 100
	if args.PerPage != nil {
		perPage = *args.PerPage
	}
	page := args.Page

	// Collect all threads.
	threads := make([]map[string]any, 0, len(s.db.Threads))
	for _, raw := range s.db.Threads {
		t, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		threads = append(threads, t)
	}

	// Apply resourceId filter.
	if args.Filter != nil && args.Filter.ResourceID != "" {
		filtered := threads[:0]
		for _, t := range threads {
			if t["resourceId"] == args.Filter.ResourceID {
				filtered = append(filtered, t)
			}
		}
		threads = filtered
	}

	// Apply metadata filter (AND logic).
	if args.Filter != nil && len(args.Filter.Metadata) > 0 {
		filtered := threads[:0]
		for _, t := range threads {
			meta, _ := t["metadata"].(map[string]any)
			if meta == nil {
				continue
			}
			allMatch := true
			for k, v := range args.Filter.Metadata {
				if !jsonValueEquals(meta[k], v) {
					allMatch = false
					break
				}
			}
			if allMatch {
				filtered = append(filtered, t)
			}
		}
		threads = filtered
	}

	// Sort.
	sortByField(threads, field, direction)

	// Clone threads (deep copy metadata).
	cloned := make([]StorageThreadType, len(threads))
	for i, t := range threads {
		cloned[i] = cloneThread(t)
	}

	total := len(cloned)
	offset := page * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}

	return StorageListThreadsOutput{
		Threads: cloned[offset:end],
		Total:   total,
		Page:    page,
		PerPage: perPage,
		HasMore: end < total,
	}, nil
}

// ---------------------------------------------------------------------------
// Message Methods
// ---------------------------------------------------------------------------

// ListMessages lists messages with optional filtering, pagination, include, and sorting.
func (s *InMemoryMemory) ListMessages(_ context.Context, args StorageListMessagesInput) (StorageListMessagesOutput, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	// Normalize threadId to slice.
	var threadIDs []string
	switch v := args.ThreadID.(type) {
	case string:
		threadIDs = []string{v}
	case []string:
		threadIDs = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				threadIDs = append(threadIDs, s)
			}
		}
	default:
		return StorageListMessagesOutput{}, fmt.Errorf("threadId must be a string or array of strings")
	}

	if len(threadIDs) == 0 {
		return StorageListMessagesOutput{}, fmt.Errorf("threadId must be a non-empty string or array of non-empty strings")
	}
	for _, id := range threadIDs {
		if strings.TrimSpace(id) == "" {
			return StorageListMessagesOutput{}, fmt.Errorf("threadId must be a non-empty string or array of non-empty strings")
		}
	}

	threadIDSet := make(map[string]bool, len(threadIDs))
	for _, id := range threadIDs {
		threadIDSet[id] = true
	}

	field, direction := parseOrderBy(args.OrderBy, "ASC")

	perPage := 40
	if args.PerPage != nil {
		perPage = *args.PerPage
	}
	page := args.Page

	if page < 0 {
		return StorageListMessagesOutput{}, fmt.Errorf("page must be >= 0")
	}

	// Collect messages matching thread(s) and optional resourceId.
	var threadMessages []map[string]any
	for _, raw := range s.db.Messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		tid, _ := msg["thread_id"].(string)
		if !threadIDSet[tid] {
			continue
		}
		if args.ResourceID != "" {
			rid, _ := msg["resourceId"].(string)
			if rid != args.ResourceID {
				continue
			}
		}
		threadMessages = append(threadMessages, msg)
	}

	// Apply date filtering.
	threadMessages = filterByDateRange(threadMessages, args.Filter)

	// Sort.
	sortByField(threadMessages, field, direction)

	totalThreadMessages := len(threadMessages)

	// Paginate.
	offset := page * perPage
	end := offset + perPage
	if end > totalThreadMessages {
		end = totalThreadMessages
	}
	if offset > totalThreadMessages {
		offset = totalThreadMessages
	}
	paginatedMsgs := threadMessages[offset:end]

	// Convert to MastraDBMessage.
	messages := make([]MastraDBMessage, 0, len(paginatedMsgs))
	messageIDs := make(map[string]bool)
	for _, msg := range paginatedMsgs {
		converted := parseStoredMessage(msg)
		messages = append(messages, converted)
		id, _ := msg["id"].(string)
		messageIDs[id] = true
	}

	// Include context messages.
	if len(args.Include) > 0 {
		for _, includeItem := range args.Include {
			targetRaw, ok := s.db.Messages[includeItem.ID]
			if !ok {
				continue
			}
			targetMsg, ok := targetRaw.(map[string]any)
			if !ok {
				continue
			}

			converted := parseStoredMessage(targetMsg)
			cid, _ := converted["id"].(string)
			if !messageIDs[cid] {
				messages = append(messages, converted)
				messageIDs[cid] = true
			}

			// Add previous messages if requested.
			if includeItem.WithPreviousMessages > 0 {
				lookupThreadID := includeItem.ThreadID
				if lookupThreadID == "" && len(threadIDs) > 0 {
					lookupThreadID = threadIDs[0]
				}
				allInThread := getMessagesByThread(s.db.Messages, lookupThreadID)
				sortByField(allInThread, "createdAt", "ASC")
				idx := findMessageIndex(allInThread, includeItem.ID)
				if idx >= 0 {
					startIdx := idx - includeItem.WithPreviousMessages
					if startIdx < 0 {
						startIdx = 0
					}
					for i := startIdx; i < idx; i++ {
						mid, _ := allInThread[i]["id"].(string)
						if !messageIDs[mid] {
							messages = append(messages, parseStoredMessage(allInThread[i]))
							messageIDs[mid] = true
						}
					}
				}
			}

			// Add next messages if requested.
			if includeItem.WithNextMessages > 0 {
				lookupThreadID := includeItem.ThreadID
				if lookupThreadID == "" && len(threadIDs) > 0 {
					lookupThreadID = threadIDs[0]
				}
				allInThread := getMessagesByThread(s.db.Messages, lookupThreadID)
				sortByField(allInThread, "createdAt", "ASC")
				idx := findMessageIndex(allInThread, includeItem.ID)
				if idx >= 0 {
					endIdx := idx + includeItem.WithNextMessages + 1
					if endIdx > len(allInThread) {
						endIdx = len(allInThread)
					}
					for i := idx + 1; i < endIdx; i++ {
						mid, _ := allInThread[i]["id"].(string)
						if !messageIDs[mid] {
							messages = append(messages, parseStoredMessage(allInThread[i]))
							messageIDs[mid] = true
						}
					}
				}
			}
		}
	}

	// Sort all messages for final output.
	sortMastraDBMessages(messages, field, direction)

	// Calculate hasMore.
	hasMore := end < totalThreadMessages

	return StorageListMessagesOutput{
		Messages: messages,
		Total:    totalThreadMessages,
		Page:     page,
		PerPage:  perPage,
		HasMore:  hasMore,
	}, nil
}

// ListMessagesByResourceID lists messages by resource ID across all threads.
func (s *InMemoryMemory) ListMessagesByResourceID(_ context.Context, args StorageListMessagesByResourceIDInput) (StorageListMessagesOutput, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	field, direction := parseOrderBy(args.OrderBy, "ASC")

	perPage := 40
	if args.PerPage != nil {
		perPage = *args.PerPage
	}
	page := args.Page

	if page < 0 {
		return StorageListMessagesOutput{}, fmt.Errorf("page must be >= 0")
	}

	// Get all messages matching the resourceId.
	var msgs []map[string]any
	for _, raw := range s.db.Messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rid, _ := msg["resourceId"].(string)
		if rid == args.ResourceID {
			msgs = append(msgs, msg)
		}
	}

	// Apply date filtering.
	msgs = filterByDateRange(msgs, args.Filter)

	// Sort.
	sortByField(msgs, field, direction)

	total := len(msgs)
	offset := page * perPage
	end := offset + perPage
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	paginated := msgs[offset:end]

	messages := make([]MastraDBMessage, 0, len(paginated))
	for _, m := range paginated {
		messages = append(messages, parseStoredMessage(m))
	}

	hasMore := offset+len(paginated) < total

	return StorageListMessagesOutput{
		Messages: messages,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
		HasMore:  hasMore,
	}, nil
}

// ListMessagesByID retrieves messages by their IDs.
func (s *InMemoryMemory) ListMessagesByID(_ context.Context, messageIDs []string) ([]MastraDBMessage, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	var messages []MastraDBMessage
	for _, id := range messageIDs {
		raw, ok := s.db.Messages[id]
		if !ok {
			continue
		}
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		messages = append(messages, parseStoredMessage(msg))
	}
	return messages, nil
}

// SaveMessages saves multiple messages.
func (s *InMemoryMemory) SaveMessages(_ context.Context, messages []MastraDBMessage) ([]MastraDBMessage, error) {
	s.db.Lock()
	defer s.db.Unlock()

	// Update thread timestamps for each unique threadId.
	threadIDs := make(map[string]bool)
	for _, msg := range messages {
		tid, _ := msg["threadId"].(string)
		if tid != "" {
			threadIDs[tid] = true
		}
	}
	for tid := range threadIDs {
		raw, ok := s.db.Threads[tid]
		if !ok {
			continue
		}
		thread, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		thread["updatedAt"] = time.Now()
	}

	for _, msg := range messages {
		id, _ := msg["id"].(string)
		if id == "" {
			continue
		}

		// Convert MastraDBMessage to StorageMessageType.
		contentJSON, _ := json.Marshal(msg["content"])
		role, _ := msg["role"].(string)
		if role == "" {
			role = "user"
		}
		typ, _ := msg["type"].(string)
		if typ == "" {
			typ = "text"
		}
		threadID, _ := msg["threadId"].(string)
		resourceID, _ := msg["resourceId"].(string)

		storageMsg := map[string]any{
			"id":         id,
			"thread_id":  threadID,
			"content":    string(contentJSON),
			"role":       role,
			"type":       typ,
			"createdAt":  msg["createdAt"],
			"resourceId": resourceID,
		}
		s.db.Messages[id] = storageMsg
	}

	return messages, nil
}

// UpdateMessages updates multiple messages with partial data (deep merge for content).
func (s *InMemoryMemory) UpdateMessages(_ context.Context, input UpdateMessagesInput) ([]MastraDBMessage, error) {
	s.db.Lock()
	defer s.db.Unlock()

	var updatedMessages []MastraDBMessage
	for _, rawUpdate := range input.Messages {
		update, ok := rawUpdate.(map[string]any)
		if !ok {
			continue
		}
		updateID, _ := update["id"].(string)
		if updateID == "" {
			continue
		}

		raw, ok := s.db.Messages[updateID]
		if !ok {
			continue
		}
		storageMsg, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		oldThreadID, _ := storageMsg["thread_id"].(string)
		newThreadID := oldThreadID
		threadIDChanged := false

		if updateTID, ok := update["threadId"].(string); ok && updateTID != "" && updateTID != oldThreadID {
			newThreadID = updateTID
			threadIDChanged = true
		}

		// Update fields.
		if v, ok := update["role"]; ok {
			storageMsg["role"] = v
		}
		if v, ok := update["type"]; ok {
			storageMsg["type"] = v
		}
		if v, ok := update["createdAt"]; ok {
			storageMsg["createdAt"] = v
		}
		if v, ok := update["resourceId"]; ok {
			storageMsg["resourceId"] = v
		}

		// Deep merge content.
		if newContent, ok := update["content"]; ok && newContent != nil {
			oldContent := safelyParseJSON(storageMsg["content"])
			oldMap, oldIsMap := oldContent.(map[string]any)
			newMap, newIsMap := newContent.(map[string]any)
			if oldIsMap && newIsMap {
				merged := make(map[string]any)
				for k, v := range oldMap {
					merged[k] = v
				}
				for k, v := range newMap {
					merged[k] = v
				}
				// Deep merge metadata if both have it.
				if oldMeta, ok := oldMap["metadata"].(map[string]any); ok {
					if newMeta, ok := newMap["metadata"].(map[string]any); ok {
						mergedMeta := make(map[string]any)
						for k, v := range oldMeta {
							mergedMeta[k] = v
						}
						for k, v := range newMeta {
							mergedMeta[k] = v
						}
						merged["metadata"] = mergedMeta
					}
				}
				contentJSON, _ := json.Marshal(merged)
				storageMsg["content"] = string(contentJSON)
			} else {
				contentJSON, _ := json.Marshal(newContent)
				storageMsg["content"] = string(contentJSON)
			}
		}

		// Handle threadId change.
		if threadIDChanged {
			storageMsg["thread_id"] = newThreadID
			base := time.Now().UnixMilli()

			oldThreadRaw, oldExists := s.db.Threads[oldThreadID]
			var oldThreadNewTime int64
			if oldExists {
				oldThread, ok := oldThreadRaw.(map[string]any)
				if ok {
					prev := getTimeMillis(oldThread["updatedAt"])
					oldThreadNewTime = max64(base, prev+1)
					oldThread["updatedAt"] = time.UnixMilli(oldThreadNewTime)
				}
			}

			newThreadRaw, newExists := s.db.Threads[newThreadID]
			if newExists {
				newThread, ok := newThreadRaw.(map[string]any)
				if ok {
					prev := getTimeMillis(newThread["updatedAt"])
					newThreadNewTime := max64(base+1, prev+1)
					if oldThreadNewTime > 0 && newThreadNewTime <= oldThreadNewTime {
						newThreadNewTime = oldThreadNewTime + 1
					}
					newThread["updatedAt"] = time.UnixMilli(newThreadNewTime)
				}
			}
		} else {
			// Update thread's updatedAt.
			threadRaw, exists := s.db.Threads[oldThreadID]
			if exists {
				thread, ok := threadRaw.(map[string]any)
				if ok {
					prev := getTimeMillis(thread["updatedAt"])
					newTime := time.Now().UnixMilli()
					if newTime <= prev {
						newTime = prev + 1
					}
					thread["updatedAt"] = time.UnixMilli(newTime)
				}
			}
		}

		s.db.Messages[updateID] = storageMsg

		// Build return value.
		role, _ := storageMsg["role"].(string)
		if role != "user" && role != "assistant" {
			role = "user"
		}
		rid := storageMsg["resourceId"]
		var ridStr string
		if rid != nil {
			ridStr, _ = rid.(string)
		}

		result := MastraDBMessage{
			"id":        storageMsg["id"],
			"threadId":  storageMsg["thread_id"],
			"content":   safelyParseJSON(storageMsg["content"]),
			"role":      role,
			"type":      storageMsg["type"],
			"createdAt": storageMsg["createdAt"],
		}
		if ridStr != "" {
			result["resourceId"] = ridStr
		}
		updatedMessages = append(updatedMessages, result)
	}

	return updatedMessages, nil
}

// DeleteMessages deletes messages by their IDs.
func (s *InMemoryMemory) DeleteMessages(_ context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	s.db.Lock()
	defer s.db.Unlock()

	threadIDs := make(map[string]bool)
	for _, mid := range messageIDs {
		raw, ok := s.db.Messages[mid]
		if ok {
			msg, ok := raw.(map[string]any)
			if ok {
				tid, _ := msg["thread_id"].(string)
				if tid != "" {
					threadIDs[tid] = true
				}
			}
		}
		delete(s.db.Messages, mid)
	}

	now := time.Now()
	for tid := range threadIDs {
		raw, ok := s.db.Threads[tid]
		if !ok {
			continue
		}
		thread, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		thread["updatedAt"] = now
	}

	return nil
}

// ---------------------------------------------------------------------------
// Resource Methods
// ---------------------------------------------------------------------------

// GetResourceByID retrieves a resource by its ID.
func (s *InMemoryMemory) GetResourceByID(_ context.Context, resourceID string) (StorageResourceType, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	raw, ok := s.db.Resources[resourceID]
	if !ok || raw == nil {
		return nil, nil
	}
	resource, ok := raw.(map[string]any)
	if !ok {
		return nil, nil
	}
	return cloneResource(resource), nil
}

// SaveResource creates or saves a resource.
func (s *InMemoryMemory) SaveResource(_ context.Context, resource StorageResourceType) (StorageResourceType, error) {
	s.db.Lock()
	defer s.db.Unlock()

	id, _ := resource["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("resource id is required")
	}
	s.db.Resources[id] = resource
	return resource, nil
}

// UpdateResource updates an existing resource (with upsert behavior).
func (s *InMemoryMemory) UpdateResource(_ context.Context, input UpdateResourceInput) (StorageResourceType, error) {
	s.db.Lock()
	defer s.db.Unlock()

	raw, exists := s.db.Resources[input.ResourceID]
	var resource map[string]any

	if !exists || raw == nil {
		// Create new resource.
		resource = map[string]any{
			"id":        input.ResourceID,
			"metadata":  input.Metadata,
			"createdAt": time.Now(),
			"updatedAt": time.Now(),
		}
		if input.WorkingMemory != nil {
			resource["workingMemory"] = *input.WorkingMemory
		}
	} else {
		existing, ok := raw.(map[string]any)
		if !ok {
			existing = map[string]any{}
		}
		resource = make(map[string]any, len(existing))
		for k, v := range existing {
			resource[k] = v
		}

		if input.WorkingMemory != nil {
			resource["workingMemory"] = *input.WorkingMemory
		}

		// Merge metadata.
		existingMeta, _ := resource["metadata"].(map[string]any)
		if existingMeta == nil {
			existingMeta = map[string]any{}
		}
		mergedMeta := make(map[string]any, len(existingMeta)+len(input.Metadata))
		for k, v := range existingMeta {
			mergedMeta[k] = v
		}
		for k, v := range input.Metadata {
			mergedMeta[k] = v
		}
		resource["metadata"] = mergedMeta
		resource["updatedAt"] = time.Now()
	}

	s.db.Resources[input.ResourceID] = resource
	return resource, nil
}

// ---------------------------------------------------------------------------
// Clone Thread
// ---------------------------------------------------------------------------

// CloneThread clones a thread and its messages to create a new independent thread.
func (s *InMemoryMemory) CloneThread(_ context.Context, args StorageCloneThreadInput) (StorageCloneThreadOutput, error) {
	s.db.Lock()
	defer s.db.Unlock()

	// Get source thread.
	sourceRaw, ok := s.db.Threads[args.SourceThreadID]
	if !ok || sourceRaw == nil {
		return StorageCloneThreadOutput{}, fmt.Errorf("source thread with id %s not found", args.SourceThreadID)
	}
	sourceThread, ok := sourceRaw.(map[string]any)
	if !ok {
		return StorageCloneThreadOutput{}, fmt.Errorf("source thread with id %s not found", args.SourceThreadID)
	}

	newThreadID := args.NewThreadID
	if newThreadID == "" {
		newThreadID = uuid.New().String()
	}

	if _, exists := s.db.Threads[newThreadID]; exists {
		return StorageCloneThreadOutput{}, fmt.Errorf("thread with id %s already exists", newThreadID)
	}

	// Get source messages sorted by createdAt.
	sourceMessages := getMessagesByThread(s.db.Messages, args.SourceThreadID)
	sortByField(sourceMessages, "createdAt", "ASC")

	// Apply message filters.
	if args.Options != nil && args.Options.MessageFilter != nil {
		mf := args.Options.MessageFilter

		if len(mf.MessageIDs) > 0 {
			idSet := make(map[string]bool, len(mf.MessageIDs))
			for _, id := range mf.MessageIDs {
				idSet[id] = true
			}
			filtered := sourceMessages[:0]
			for _, msg := range sourceMessages {
				id, _ := msg["id"].(string)
				if idSet[id] {
					filtered = append(filtered, msg)
				}
			}
			sourceMessages = filtered
		}

		if mf.StartDate != nil {
			filtered := sourceMessages[:0]
			for _, msg := range sourceMessages {
				createdAt := getTime(msg["createdAt"])
				if !createdAt.Before(*mf.StartDate) {
					filtered = append(filtered, msg)
				}
			}
			sourceMessages = filtered
		}

		if mf.EndDate != nil {
			filtered := sourceMessages[:0]
			for _, msg := range sourceMessages {
				createdAt := getTime(msg["createdAt"])
				if !createdAt.After(*mf.EndDate) {
					filtered = append(filtered, msg)
				}
			}
			sourceMessages = filtered
		}
	}

	// Apply message limit (take from the end for most recent).
	if args.Options != nil && args.Options.MessageLimit > 0 && len(sourceMessages) > args.Options.MessageLimit {
		sourceMessages = sourceMessages[len(sourceMessages)-args.Options.MessageLimit:]
	}

	now := time.Now()

	// Determine last message ID.
	var lastMessageID string
	if len(sourceMessages) > 0 {
		lastMessageID, _ = sourceMessages[len(sourceMessages)-1]["id"].(string)
	}

	// Create clone metadata.
	cloneMetadata := map[string]any{
		"sourceThreadId": args.SourceThreadID,
		"clonedAt":       now,
	}
	if lastMessageID != "" {
		cloneMetadata["lastMessageId"] = lastMessageID
	}

	// Build merged metadata.
	mergedMetadata := make(map[string]any)
	for k, v := range args.Metadata {
		mergedMetadata[k] = v
	}
	mergedMetadata["clone"] = cloneMetadata

	// Determine resourceId and title.
	resourceID := args.ResourceID
	if resourceID == "" {
		resourceID, _ = sourceThread["resourceId"].(string)
	}
	title := args.Title
	if title == "" {
		sourceTitle, _ := sourceThread["title"].(string)
		if sourceTitle != "" {
			title = "Clone of " + sourceTitle
		}
	}

	// Create new thread.
	newThread := StorageThreadType{
		"id":         newThreadID,
		"resourceId": resourceID,
		"title":      title,
		"metadata":   mergedMetadata,
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.db.Threads[newThreadID] = newThread

	// Clone messages with new IDs.
	clonedMessages := make([]MastraDBMessage, 0, len(sourceMessages))
	messageIDMap := make(map[string]string)

	for _, srcMsg := range sourceMessages {
		newMsgID := uuid.New().String()
		srcID, _ := srcMsg["id"].(string)
		messageIDMap[srcID] = newMsgID

		cloneResourceID := resourceID
		if cloneResourceID == "" {
			cloneResourceID, _ = srcMsg["resourceId"].(string)
		}

		// Create storage message.
		newStorageMsg := map[string]any{
			"id":         newMsgID,
			"thread_id":  newThreadID,
			"content":    srcMsg["content"],
			"role":       srcMsg["role"],
			"type":       srcMsg["type"],
			"createdAt":  srcMsg["createdAt"],
			"resourceId": cloneResourceID,
		}
		s.db.Messages[newMsgID] = newStorageMsg

		// Create MastraDBMessage for return.
		parsedContent := safelyParseJSON(srcMsg["content"])
		clonedMsg := MastraDBMessage{
			"id":        newMsgID,
			"threadId":  newThreadID,
			"content":   parsedContent,
			"role":      srcMsg["role"],
			"type":      srcMsg["type"],
			"createdAt": srcMsg["createdAt"],
		}
		if cloneResourceID != "" {
			clonedMsg["resourceId"] = cloneResourceID
		}
		clonedMessages = append(clonedMessages, clonedMsg)
	}

	return StorageCloneThreadOutput{
		Thread:         newThread,
		ClonedMessages: clonedMessages,
		MessageIDMap:   messageIDMap,
	}, nil
}

// ---------------------------------------------------------------------------
// Observational Memory Implementation
// ---------------------------------------------------------------------------

func (s *InMemoryMemory) getObservationalMemoryKey(threadID, resourceID string) string {
	if threadID != "" {
		return "thread:" + threadID
	}
	return "resource:" + resourceID
}

// GetObservationalMemory gets the current (most recent) observational memory record.
func (s *InMemoryMemory) GetObservationalMemory(_ context.Context, threadID, resourceID string) (*ObservationalMemoryRecord, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	key := s.getObservationalMemoryKey(threadID, resourceID)
	records := s.db.ObservationalMemory[key]
	if len(records) == 0 {
		return nil, nil
	}
	rec, ok := records[0].(*ObservationalMemoryRecord)
	if !ok {
		return nil, nil
	}
	return rec, nil
}

// GetObservationalMemoryHistory gets observational memory history.
func (s *InMemoryMemory) GetObservationalMemoryHistory(_ context.Context, threadID, resourceID string, limit int) ([]ObservationalMemoryRecord, error) {
	s.db.RLock()
	defer s.db.RUnlock()

	key := s.getObservationalMemoryKey(threadID, resourceID)
	records := s.db.ObservationalMemory[key]

	result := make([]ObservationalMemoryRecord, 0, len(records))
	for _, raw := range records {
		rec, ok := raw.(*ObservationalMemoryRecord)
		if !ok {
			continue
		}
		result = append(result, *rec)
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// InitializeObservationalMemory creates a new observational memory record.
func (s *InMemoryMemory) InitializeObservationalMemory(_ context.Context, input CreateObservationalMemoryInput) (*ObservationalMemoryRecord, error) {
	s.db.Lock()
	defer s.db.Unlock()

	key := s.getObservationalMemoryKey(input.ThreadID, input.ResourceID)
	now := time.Now()

	record := &ObservationalMemoryRecord{
		ID:                    uuid.New().String(),
		Scope:                 input.Scope,
		ThreadID:              input.ThreadID,
		ResourceID:            input.ResourceID,
		CreatedAt:             now,
		UpdatedAt:             now,
		LastObservedAt:        nil,
		OriginType:            ObservationalMemoryOriginInitial,
		GenerationCount:       0,
		ActiveObservations:    "",
		TotalTokensObserved:   0,
		ObservationTokenCount: 0,
		PendingMessageTokens:  0,
		IsReflecting:          false,
		IsObserving:           false,
		IsBufferingObservation: false,
		IsBufferingReflection: false,
		LastBufferedAtTokens:  0,
		LastBufferedAtTime:    nil,
		Config:                input.Config,
		ObservedTimezone:      input.ObservedTimezone,
		Metadata:              map[string]any{},
	}

	existing := s.db.ObservationalMemory[key]
	// Prepend (most recent first).
	newRecords := make([]any, 0, len(existing)+1)
	newRecords = append(newRecords, record)
	newRecords = append(newRecords, existing...)
	s.db.ObservationalMemory[key] = newRecords

	return record, nil
}

// InsertObservationalMemoryRecord inserts a fully-formed record (used by thread cloning).
func (s *InMemoryMemory) InsertObservationalMemoryRecord(_ context.Context, record ObservationalMemoryRecord) error {
	s.db.Lock()
	defer s.db.Unlock()

	key := s.getObservationalMemoryKey(record.ThreadID, record.ResourceID)
	existing := s.db.ObservationalMemory[key]

	// Insert in order by generationCount descending (newest first).
	inserted := false
	newRecords := make([]any, 0, len(existing)+1)
	rec := record // copy
	for i, raw := range existing {
		existingRec, ok := raw.(*ObservationalMemoryRecord)
		if ok && !inserted && record.GenerationCount >= existingRec.GenerationCount {
			newRecords = append(newRecords, &rec)
			inserted = true
			newRecords = append(newRecords, existing[i:]...)
			break
		}
		newRecords = append(newRecords, raw)
	}
	if !inserted {
		newRecords = append(newRecords, &rec)
	}
	s.db.ObservationalMemory[key] = newRecords

	return nil
}

// UpdateActiveObservations updates active observations.
func (s *InMemoryMemory) UpdateActiveObservations(_ context.Context, input UpdateActiveObservationsInput) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(input.ID)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", input.ID)
	}

	record.ActiveObservations = input.Observations
	record.ObservationTokenCount = input.TokenCount
	record.TotalTokensObserved += input.TokenCount
	record.PendingMessageTokens = 0
	record.LastObservedAt = &input.LastObservedAt
	record.UpdatedAt = time.Now()

	if len(input.ObservedMessageIDs) > 0 {
		record.ObservedMessageIDs = input.ObservedMessageIDs
	}

	return nil
}

// UpdateBufferedObservations updates buffered observations.
func (s *InMemoryMemory) UpdateBufferedObservations(_ context.Context, input UpdateBufferedObservationsInput) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(input.ID)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", input.ID)
	}

	newChunk := BufferedObservationChunk{
		ID:                    "ombuf-" + uuid.New().String(),
		CycleID:               input.Chunk.CycleID,
		Observations:          input.Chunk.Observations,
		TokenCount:            input.Chunk.TokenCount,
		MessageIDs:            input.Chunk.MessageIDs,
		MessageTokens:         input.Chunk.MessageTokens,
		LastObservedAt:        input.Chunk.LastObservedAt,
		CreatedAt:             time.Now(),
		SuggestedContinuation: input.Chunk.SuggestedContinuation,
		CurrentTask:           input.Chunk.CurrentTask,
	}

	record.BufferedObservationChunks = append(record.BufferedObservationChunks, newChunk)

	if input.LastBufferedAtTime != nil {
		record.LastBufferedAtTime = input.LastBufferedAtTime
	}

	record.UpdatedAt = time.Now()
	return nil
}

// SwapBufferedToActive swaps buffered observations to active using the activation ratio algorithm.
func (s *InMemoryMemory) SwapBufferedToActive(_ context.Context, input SwapBufferedToActiveInput) (*SwapBufferedToActiveResult, error) {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(input.ID)
	if record == nil {
		return nil, fmt.Errorf("observational memory record not found: %s", input.ID)
	}

	chunks := record.BufferedObservationChunks
	if len(chunks) == 0 {
		return &SwapBufferedToActiveResult{
			ChunksActivated:            0,
			MessageTokensActivated:     0,
			ObservationTokensActivated: 0,
			MessagesActivated:          0,
			ActivatedCycleIDs:          []string{},
			ActivatedMessageIDs:        []string{},
		}, nil
	}

	// Calculate target: how many message tokens to remove so that
	// (1 - activationRatio) * threshold worth of raw messages remain.
	retentionFloor := float64(input.MessageTokensThreshold) * (1 - input.ActivationRatio)
	targetMessageTokens := math.Max(0, float64(input.CurrentPendingTokens)-retentionFloor)

	// Find the closest chunk boundary to the target.
	var cumulativeMessageTokens float64
	var bestOverBoundary, bestUnderBoundary int
	var bestOverTokens, bestUnderTokens float64

	for i := 0; i < len(chunks); i++ {
		cumulativeMessageTokens += float64(chunks[i].MessageTokens)
		boundary := i + 1

		if cumulativeMessageTokens >= targetMessageTokens {
			if bestOverBoundary == 0 || cumulativeMessageTokens < bestOverTokens {
				bestOverBoundary = boundary
				bestOverTokens = cumulativeMessageTokens
			}
		} else {
			if cumulativeMessageTokens > bestUnderTokens {
				bestUnderBoundary = boundary
				bestUnderTokens = cumulativeMessageTokens
			}
		}
	}

	// Safeguard logic.
	maxOvershoot := retentionFloor * 0.95
	overshoot := bestOverTokens - targetMessageTokens
	remainingAfterOver := float64(input.CurrentPendingTokens) - bestOverTokens
	remainingAfterUnder := float64(input.CurrentPendingTokens) - bestUnderTokens
	minRemaining := math.Min(1000, retentionFloor)

	var chunksToActivate int
	if input.ForceMaxActivation && bestOverBoundary > 0 && remainingAfterOver >= minRemaining {
		chunksToActivate = bestOverBoundary
	} else if bestOverBoundary > 0 && overshoot <= maxOvershoot && remainingAfterOver >= minRemaining {
		chunksToActivate = bestOverBoundary
	} else if bestUnderBoundary > 0 && remainingAfterUnder >= minRemaining {
		chunksToActivate = bestUnderBoundary
	} else if bestOverBoundary > 0 {
		chunksToActivate = bestOverBoundary
	} else {
		chunksToActivate = 1
	}

	activatedChunks := chunks[:chunksToActivate]
	remainingChunks := chunks[chunksToActivate:]

	// Combine activated chunks.
	var observationParts []string
	var activatedTokens, activatedMessageTokens, activatedMessageCount int
	var activatedCycleIDs, activatedMessageIDs []string

	for _, c := range activatedChunks {
		observationParts = append(observationParts, c.Observations)
		activatedTokens += c.TokenCount
		activatedMessageTokens += c.MessageTokens
		activatedMessageCount += len(c.MessageIDs)
		if c.CycleID != "" {
			activatedCycleIDs = append(activatedCycleIDs, c.CycleID)
		}
		activatedMessageIDs = append(activatedMessageIDs, c.MessageIDs...)
	}

	activatedContent := strings.Join(observationParts, "\n\n")

	// Derive lastObservedAt.
	latestChunk := activatedChunks[len(activatedChunks)-1]
	derivedLastObservedAt := input.LastObservedAt
	if derivedLastObservedAt == nil {
		t := latestChunk.LastObservedAt
		derivedLastObservedAt = &t
	}

	// Append activated content to active observations.
	if record.ActiveObservations != "" {
		record.ActiveObservations = record.ActiveObservations + "\n\n" + activatedContent
	} else {
		record.ActiveObservations = activatedContent
	}

	// Update observation token count.
	record.ObservationTokenCount += activatedTokens

	// Decrement pending message tokens (clamped to zero).
	record.PendingMessageTokens -= activatedMessageTokens
	if record.PendingMessageTokens < 0 {
		record.PendingMessageTokens = 0
	}

	// Update buffered state with remaining chunks.
	if len(remainingChunks) > 0 {
		record.BufferedObservationChunks = remainingChunks
	} else {
		record.BufferedObservationChunks = nil
	}

	// Update timestamps.
	record.LastObservedAt = derivedLastObservedAt
	record.UpdatedAt = time.Now()

	// Build per-chunk breakdown.
	perChunk := make([]SwapBufferedToActivePerChunk, len(activatedChunks))
	for i, c := range activatedChunks {
		cycleID := c.CycleID
		perChunk[i] = SwapBufferedToActivePerChunk{
			CycleID:           cycleID,
			MessageTokens:     c.MessageTokens,
			ObservationTokens: c.TokenCount,
			MessageCount:      len(c.MessageIDs),
			Observations:      c.Observations,
		}
	}

	// Use hints from the most recent activated chunk only.
	latestHints := activatedChunks[len(activatedChunks)-1]

	return &SwapBufferedToActiveResult{
		ChunksActivated:            len(activatedChunks),
		MessageTokensActivated:     activatedMessageTokens,
		ObservationTokensActivated: activatedTokens,
		MessagesActivated:          activatedMessageCount,
		ActivatedCycleIDs:          activatedCycleIDs,
		ActivatedMessageIDs:        activatedMessageIDs,
		Observations:               activatedContent,
		PerChunk:                   perChunk,
		SuggestedContinuation:      latestHints.SuggestedContinuation,
		CurrentTask:                latestHints.CurrentTask,
	}, nil
}

// CreateReflectionGeneration creates a new generation from a reflection.
func (s *InMemoryMemory) CreateReflectionGeneration(_ context.Context, input CreateReflectionGenerationInput) (*ObservationalMemoryRecord, error) {
	s.db.Lock()
	defer s.db.Unlock()

	return s.createReflectionGenerationLocked(input)
}

func (s *InMemoryMemory) createReflectionGenerationLocked(input CreateReflectionGenerationInput) (*ObservationalMemoryRecord, error) {
	currentRecord := input.CurrentRecord
	key := s.getObservationalMemoryKey(currentRecord.ThreadID, currentRecord.ResourceID)
	now := time.Now()

	lastObservedAt := currentRecord.LastObservedAt
	if lastObservedAt == nil {
		lastObservedAt = &now
	}

	newRecord := &ObservationalMemoryRecord{
		ID:                     uuid.New().String(),
		Scope:                  currentRecord.Scope,
		ThreadID:               currentRecord.ThreadID,
		ResourceID:             currentRecord.ResourceID,
		CreatedAt:              now,
		UpdatedAt:              now,
		LastObservedAt:         lastObservedAt,
		OriginType:             ObservationalMemoryOriginReflection,
		GenerationCount:        currentRecord.GenerationCount + 1,
		ActiveObservations:     input.Reflection,
		Config:                 currentRecord.Config,
		TotalTokensObserved:    currentRecord.TotalTokensObserved,
		ObservationTokenCount:  input.TokenCount,
		PendingMessageTokens:   0,
		IsReflecting:           false,
		IsObserving:            false,
		IsBufferingObservation: false,
		IsBufferingReflection:  false,
		LastBufferedAtTokens:   0,
		LastBufferedAtTime:     nil,
		ObservedTimezone:       currentRecord.ObservedTimezone,
		Metadata:               map[string]any{},
	}

	// Add as first record (most recent).
	existing := s.db.ObservationalMemory[key]
	newRecords := make([]any, 0, len(existing)+1)
	newRecords = append(newRecords, newRecord)
	newRecords = append(newRecords, existing...)
	s.db.ObservationalMemory[key] = newRecords

	return newRecord, nil
}

// UpdateBufferedReflection updates the buffered reflection.
func (s *InMemoryMemory) UpdateBufferedReflection(_ context.Context, input UpdateBufferedReflectionInput) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(input.ID)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", input.ID)
	}

	existing := record.BufferedReflection
	if existing != "" {
		record.BufferedReflection = existing + "\n\n" + input.Reflection
	} else {
		record.BufferedReflection = input.Reflection
	}

	bufferedTokens := 0
	if record.BufferedReflectionTokens != nil {
		bufferedTokens = *record.BufferedReflectionTokens
	}
	bufferedTokens += input.TokenCount
	record.BufferedReflectionTokens = &bufferedTokens

	bufferedInputTokens := 0
	if record.BufferedReflectionInputTokens != nil {
		bufferedInputTokens = *record.BufferedReflectionInputTokens
	}
	bufferedInputTokens += input.InputTokenCount
	record.BufferedReflectionInputTokens = &bufferedInputTokens

	lineCount := input.ReflectedObservationLineCount
	record.ReflectedObservationLineCount = &lineCount
	record.UpdatedAt = time.Now()

	return nil
}

// SwapBufferedReflectionToActive swaps buffered reflection to active observations.
func (s *InMemoryMemory) SwapBufferedReflectionToActive(_ context.Context, input SwapBufferedReflectionToActiveInput) (*ObservationalMemoryRecord, error) {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(input.CurrentRecord.ID)
	if record == nil {
		return nil, fmt.Errorf("observational memory record not found: %s", input.CurrentRecord.ID)
	}

	if record.BufferedReflection == "" {
		return nil, fmt.Errorf("no buffered reflection to swap")
	}

	bufferedReflection := record.BufferedReflection
	reflectedLineCount := 0
	if record.ReflectedObservationLineCount != nil {
		reflectedLineCount = *record.ReflectedObservationLineCount
	}

	// Split current activeObservations by the boundary line count.
	currentObservations := record.ActiveObservations
	allLines := strings.Split(currentObservations, "\n")
	var unreflectedLines []string
	if reflectedLineCount < len(allLines) {
		unreflectedLines = allLines[reflectedLineCount:]
	}
	unreflectedContent := strings.TrimSpace(strings.Join(unreflectedLines, "\n"))

	// New activeObservations = bufferedReflection + unreflected observations.
	var newObservations string
	if unreflectedContent != "" {
		newObservations = bufferedReflection + "\n\n" + unreflectedContent
	} else {
		newObservations = bufferedReflection
	}

	newRecord, err := s.createReflectionGenerationLocked(CreateReflectionGenerationInput{
		CurrentRecord: record,
		Reflection:    newObservations,
		TokenCount:    input.TokenCount,
	})
	if err != nil {
		return nil, err
	}

	// Clear buffered state on old record.
	record.BufferedReflection = ""
	record.BufferedReflectionTokens = nil
	record.BufferedReflectionInputTokens = nil
	record.ReflectedObservationLineCount = nil

	return newRecord, nil
}

// SetReflectingFlag sets the isReflecting flag.
func (s *InMemoryMemory) SetReflectingFlag(_ context.Context, id string, isReflecting bool) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(id)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", id)
	}
	record.IsReflecting = isReflecting
	record.UpdatedAt = time.Now()
	return nil
}

// SetObservingFlag sets the isObserving flag.
func (s *InMemoryMemory) SetObservingFlag(_ context.Context, id string, isObserving bool) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(id)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", id)
	}
	record.IsObserving = isObserving
	record.UpdatedAt = time.Now()
	return nil
}

// SetBufferingObservationFlag sets the isBufferingObservation flag.
func (s *InMemoryMemory) SetBufferingObservationFlag(_ context.Context, id string, isBuffering bool, lastBufferedAtTokens *int) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(id)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", id)
	}
	record.IsBufferingObservation = isBuffering
	if lastBufferedAtTokens != nil {
		record.LastBufferedAtTokens = *lastBufferedAtTokens
	}
	record.UpdatedAt = time.Now()
	return nil
}

// SetBufferingReflectionFlag sets the isBufferingReflection flag.
func (s *InMemoryMemory) SetBufferingReflectionFlag(_ context.Context, id string, isBuffering bool) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(id)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", id)
	}
	record.IsBufferingReflection = isBuffering
	record.UpdatedAt = time.Now()
	return nil
}

// ClearObservationalMemory clears all observational memory for a thread/resource.
func (s *InMemoryMemory) ClearObservationalMemory(_ context.Context, threadID, resourceID string) error {
	s.db.Lock()
	defer s.db.Unlock()

	key := s.getObservationalMemoryKey(threadID, resourceID)
	delete(s.db.ObservationalMemory, key)
	return nil
}

// SetPendingMessageTokens sets the pending message token count.
func (s *InMemoryMemory) SetPendingMessageTokens(_ context.Context, id string, tokenCount int) error {
	s.db.Lock()
	defer s.db.Unlock()

	record := s.findObservationalMemoryRecordByID(id)
	if record == nil {
		return fmt.Errorf("observational memory record not found: %s", id)
	}
	record.PendingMessageTokens = tokenCount
	record.UpdatedAt = time.Now()
	return nil
}

// findObservationalMemoryRecordByID finds a record by ID across all keys.
// Must be called with the lock held.
func (s *InMemoryMemory) findObservationalMemoryRecordByID(id string) *ObservationalMemoryRecord {
	for _, records := range s.db.ObservationalMemory {
		for _, raw := range records {
			rec, ok := raw.(*ObservationalMemoryRecord)
			if ok && rec.ID == id {
				return rec
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseOrderBy extracts field and direction from an OrderBy config.
func parseOrderBy(orderBy *StorageOrderBy, defaultDirection string) (string, string) {
	field := "createdAt"
	direction := defaultDirection

	if orderBy != nil {
		if orderBy.Field == "createdAt" || orderBy.Field == "updatedAt" {
			field = orderBy.Field
		}
		if orderBy.Direction == "ASC" || orderBy.Direction == "DESC" {
			direction = orderBy.Direction
		}
	}
	return field, direction
}

// parseStoredMessage converts a StorageMessageType to MastraDBMessage.
func parseStoredMessage(msg map[string]any) MastraDBMessage {
	content := safelyParseJSON(msg["content"])

	// If the result is a plain string (V1 format), wrap it in V2 structure.
	if s, ok := content.(string); ok {
		content = map[string]any{
			"format":  2,
			"content": s,
			"parts":   []any{map[string]any{"type": "text", "text": s}},
		}
	}

	result := MastraDBMessage{
		"id":        msg["id"],
		"threadId":  msg["thread_id"],
		"content":   content,
		"role":      msg["role"],
		"type":      msg["type"],
		"createdAt": msg["createdAt"],
	}

	if rid, ok := msg["resourceId"].(string); ok && rid != "" {
		result["resourceId"] = rid
	}

	return result
}

// safelyParseJSON attempts to parse a JSON string, returning the original value on failure.
func safelyParseJSON(input any) any {
	if input == nil {
		return map[string]any{}
	}
	s, ok := input.(string)
	if !ok {
		return input
	}
	var parsed any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return s
	}
	return parsed
}

// cloneThread creates a shallow copy of a thread map with cloned metadata.
func cloneThread(t map[string]any) map[string]any {
	result := make(map[string]any, len(t))
	for k, v := range t {
		result[k] = v
	}
	if meta, ok := t["metadata"].(map[string]any); ok {
		clonedMeta := make(map[string]any, len(meta))
		for k, v := range meta {
			clonedMeta[k] = v
		}
		result["metadata"] = clonedMeta
	}
	return result
}

// cloneResource creates a shallow copy of a resource map with cloned metadata.
func cloneResource(r map[string]any) map[string]any {
	result := make(map[string]any, len(r))
	for k, v := range r {
		result[k] = v
	}
	if meta, ok := r["metadata"].(map[string]any); ok {
		clonedMeta := make(map[string]any, len(meta))
		for k, v := range meta {
			clonedMeta[k] = v
		}
		result["metadata"] = clonedMeta
	}
	return result
}

// sortByField sorts a slice of maps by a field in the given direction.
func sortByField(items []map[string]any, field, direction string) {
	sort.SliceStable(items, func(i, j int) bool {
		aVal := items[i][field]
		bVal := items[j][field]
		cmp := compareValues(aVal, bVal, field)
		if direction == "DESC" {
			return cmp > 0
		}
		return cmp < 0
	})
}

// sortMastraDBMessages sorts MastraDBMessage slices by a field.
func sortMastraDBMessages(items []MastraDBMessage, field, direction string) {
	sort.SliceStable(items, func(i, j int) bool {
		aVal := items[i][field]
		bVal := items[j][field]
		cmp := compareValues(aVal, bVal, field)
		if direction == "DESC" {
			return cmp > 0
		}
		return cmp < 0
	})
}

// compareValues compares two values for sorting purposes.
func compareValues(a, b any, field string) int {
	isDateField := field == "createdAt" || field == "updatedAt"

	if isDateField {
		aTime := getTimeMillis(a)
		bTime := getTimeMillis(b)
		if aTime < bTime {
			return -1
		}
		if aTime > bTime {
			return 1
		}
		return 0
	}

	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.Compare(aStr, bStr)
}

// getTime converts any time representation to time.Time.
func getTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case *time.Time:
		if t != nil {
			return *t
		}
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// getTimeMillis converts any time representation to milliseconds.
func getTimeMillis(v any) int64 {
	return getTime(v).UnixMilli()
}

// toFloat64 tries to convert a value to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	default:
		return 0, false
	}
}

// max64 returns the larger of two int64 values.
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// getMessagesByThread returns all messages for a given thread ID.
func getMessagesByThread(messages map[string]any, threadID string) []map[string]any {
	var result []map[string]any
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		tid, _ := msg["thread_id"].(string)
		if tid == threadID {
			result = append(result, msg)
		}
	}
	return result
}

// findMessageIndex finds the index of a message by ID in a sorted slice.
func findMessageIndex(messages []map[string]any, id string) int {
	for i, msg := range messages {
		mid, _ := msg["id"].(string)
		if mid == id {
			return i
		}
	}
	return -1
}

// filterByDateRange filters messages by a date range filter.
func filterByDateRange(messages []map[string]any, filter *MessagesFilter) []map[string]any {
	if filter == nil || filter.DateRange == nil {
		return messages
	}
	dr := filter.DateRange

	result := messages
	if dr.Start != nil {
		startTime := dr.Start.UnixNano()
		filtered := make([]map[string]any, 0, len(result))
		for _, msg := range result {
			t := getTime(msg["createdAt"]).UnixNano()
			if dr.StartExclusive {
				if t > startTime {
					filtered = append(filtered, msg)
				}
			} else {
				if t >= startTime {
					filtered = append(filtered, msg)
				}
			}
		}
		result = filtered
	}

	if dr.End != nil {
		endTime := dr.End.UnixNano()
		filtered := make([]map[string]any, 0, len(result))
		for _, msg := range result {
			t := getTime(msg["createdAt"]).UnixNano()
			if dr.EndExclusive {
				if t < endTime {
					filtered = append(filtered, msg)
				}
			} else {
				if t <= endTime {
					filtered = append(filtered, msg)
				}
			}
		}
		result = filtered
	}

	return result
}

// jsonValueEquals performs a deep equality comparison on JSON-compatible values.
func jsonValueEquals(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
