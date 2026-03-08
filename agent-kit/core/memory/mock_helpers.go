// Ported from: packages/core/src/memory/mock.ts (helper conversions)
package memory

import (
	"time"

	memorystorage "github.com/brainlet/brainkit/agent-kit/core/storage/domains/memory"
)

// ---------------------------------------------------------------------------
// Conversion helpers between memory package typed structs and storage domain
// map[string]any types.
//
// The memory package defines StorageThreadType as a struct with typed fields
// (ID, Title, ResourceID, CreatedAt, UpdatedAt, Metadata), while the storage
// domain defines StorageThreadType = map[string]any.
//
// Similarly, StorageListThreadsInput, StorageCloneThreadInput, etc. are
// map[string]any aliases in the memory package but concrete structs in the
// storage domain.
//
// These helpers bridge between the two representations.
// ---------------------------------------------------------------------------

// mapToStorageThreadType converts a storage domain thread (map[string]any) to
// the memory package's typed StorageThreadType struct.
func mapToStorageThreadType(m memorystorage.StorageThreadType) StorageThreadType {
	thread := StorageThreadType{}
	if m == nil {
		return thread
	}

	if id, ok := m["id"].(string); ok {
		thread.ID = id
	}
	if title, ok := m["title"].(string); ok {
		thread.Title = title
	}
	if resourceID, ok := m["resourceId"].(string); ok {
		thread.ResourceID = resourceID
	}
	if createdAt, ok := m["createdAt"].(time.Time); ok {
		thread.CreatedAt = createdAt
	}
	if updatedAt, ok := m["updatedAt"].(time.Time); ok {
		thread.UpdatedAt = updatedAt
	}
	if metadata, ok := m["metadata"].(map[string]any); ok {
		thread.Metadata = metadata
	}
	return thread
}

// storageThreadTypeToMap converts the memory package's typed StorageThreadType
// struct to a storage domain thread (map[string]any).
func storageThreadTypeToMap(thread StorageThreadType) memorystorage.StorageThreadType {
	m := map[string]any{
		"id":         thread.ID,
		"resourceId": thread.ResourceID,
		"createdAt":  thread.CreatedAt,
		"updatedAt":  thread.UpdatedAt,
	}
	if thread.Title != "" {
		m["title"] = thread.Title
	}
	if thread.Metadata != nil {
		m["metadata"] = thread.Metadata
	}
	return m
}

// mapToStorageListThreadsInput converts the memory package's
// StorageListThreadsInput (map[string]any) to the storage domain's typed struct.
func mapToStorageListThreadsInput(args StorageListThreadsInput) memorystorage.StorageListThreadsInput {
	result := memorystorage.StorageListThreadsInput{}

	if args == nil {
		return result
	}

	if perPage, ok := args["perPage"].(int); ok {
		result.PerPage = &perPage
	}
	if page, ok := args["page"].(int); ok {
		result.Page = page
	}
	if orderBy, ok := args["orderBy"].(map[string]any); ok {
		ob := &memorystorage.StorageOrderBy{}
		if field, ok := orderBy["field"].(string); ok {
			ob.Field = field
		}
		if dir, ok := orderBy["direction"].(string); ok {
			ob.Direction = dir
		}
		result.OrderBy = ob
	}
	if filter, ok := args["filter"].(map[string]any); ok {
		f := &memorystorage.ThreadsFilter{}
		if resourceID, ok := filter["resourceId"].(string); ok {
			f.ResourceID = resourceID
		}
		if metadata, ok := filter["metadata"].(map[string]any); ok {
			f.Metadata = metadata
		}
		result.Filter = f
	}
	// Also handle top-level resourceId for convenience
	if resourceID, ok := args["resourceId"].(string); ok && resourceID != "" {
		if result.Filter == nil {
			result.Filter = &memorystorage.ThreadsFilter{}
		}
		result.Filter.ResourceID = resourceID
	}

	return result
}

// storageListThreadsOutputToMap converts the storage domain's typed
// StorageListThreadsOutput struct to the memory package's map[string]any.
func storageListThreadsOutputToMap(output memorystorage.StorageListThreadsOutput) StorageListThreadsOutput {
	threads := make([]map[string]any, len(output.Threads))
	for i, t := range output.Threads {
		threads[i] = t
	}
	return map[string]any{
		"threads": threads,
		"total":   output.Total,
		"page":    output.Page,
		"perPage": output.PerPage,
		"hasMore": output.HasMore,
	}
}

// mapToStorageListMessagesInput converts the memory package's
// StorageListMessagesInput (map[string]any) to the storage domain's typed struct.
func mapToStorageListMessagesInput(args StorageListMessagesInput) memorystorage.StorageListMessagesInput {
	result := memorystorage.StorageListMessagesInput{}

	if args == nil {
		return result
	}

	// threadId can be string or []string
	result.ThreadID = args["threadId"]

	if resourceID, ok := args["resourceId"].(string); ok {
		result.ResourceID = resourceID
	}
	if perPage, ok := args["perPage"].(int); ok {
		result.PerPage = &perPage
	}
	if page, ok := args["page"].(int); ok {
		result.Page = page
	}
	if orderBy, ok := args["orderBy"].(map[string]any); ok {
		ob := &memorystorage.StorageOrderBy{}
		if field, ok := orderBy["field"].(string); ok {
			ob.Field = field
		}
		if dir, ok := orderBy["direction"].(string); ok {
			ob.Direction = dir
		}
		result.OrderBy = ob
	}
	if filter, ok := args["filter"].(map[string]any); ok {
		f := &memorystorage.MessagesFilter{}
		if dateRange, ok := filter["dateRange"].(map[string]any); ok {
			dr := &memorystorage.DateRangeFilter{}
			if start, ok := dateRange["start"].(time.Time); ok {
				dr.Start = &start
			}
			if end, ok := dateRange["end"].(time.Time); ok {
				dr.End = &end
			}
			if startExclusive, ok := dateRange["startExclusive"].(bool); ok {
				dr.StartExclusive = startExclusive
			}
			if endExclusive, ok := dateRange["endExclusive"].(bool); ok {
				dr.EndExclusive = endExclusive
			}
			f.DateRange = dr
		}
		result.Filter = f
	}
	if include, ok := args["include"].([]any); ok {
		for _, item := range include {
			if itemMap, ok := item.(map[string]any); ok {
				mi := memorystorage.MessageIncludeItem{}
				if id, ok := itemMap["id"].(string); ok {
					mi.ID = id
				}
				if threadID, ok := itemMap["threadId"].(string); ok {
					mi.ThreadID = threadID
				}
				if prev, ok := itemMap["withPreviousMessages"].(int); ok {
					mi.WithPreviousMessages = prev
				}
				if next, ok := itemMap["withNextMessages"].(int); ok {
					mi.WithNextMessages = next
				}
				result.Include = append(result.Include, mi)
			}
		}
	}

	return result
}

// mapToStorageCloneThreadInput converts the memory package's
// StorageCloneThreadInput (map[string]any) to the storage domain's typed struct.
func mapToStorageCloneThreadInput(args StorageCloneThreadInput) memorystorage.StorageCloneThreadInput {
	result := memorystorage.StorageCloneThreadInput{}

	if args == nil {
		return result
	}

	if sourceThreadID, ok := args["sourceThreadId"].(string); ok {
		result.SourceThreadID = sourceThreadID
	}
	if newThreadID, ok := args["newThreadId"].(string); ok {
		result.NewThreadID = newThreadID
	}
	if resourceID, ok := args["resourceId"].(string); ok {
		result.ResourceID = resourceID
	}
	if title, ok := args["title"].(string); ok {
		result.Title = title
	}
	if metadata, ok := args["metadata"].(map[string]any); ok {
		result.Metadata = metadata
	}
	if options, ok := args["options"].(map[string]any); ok {
		opts := &memorystorage.CloneThreadOptions{}
		if limit, ok := options["messageLimit"].(int); ok {
			opts.MessageLimit = limit
		}
		if filter, ok := options["messageFilter"].(map[string]any); ok {
			f := &memorystorage.CloneMessageFilter{}
			if startDate, ok := filter["startDate"].(time.Time); ok {
				f.StartDate = &startDate
			}
			if endDate, ok := filter["endDate"].(time.Time); ok {
				f.EndDate = &endDate
			}
			if messageIDs, ok := filter["messageIds"].([]string); ok {
				f.MessageIDs = messageIDs
			}
			opts.MessageFilter = f
		}
		result.Options = opts
	}

	return result
}

// storageCloneThreadOutputToMap converts the storage domain's typed
// StorageCloneThreadOutput struct to the memory package's map[string]any.
func storageCloneThreadOutputToMap(output memorystorage.StorageCloneThreadOutput) StorageCloneThreadOutput {
	return map[string]any{
		"thread":         output.Thread,
		"clonedMessages": output.ClonedMessages,
		"messageIdMap":   output.MessageIDMap,
	}
}
